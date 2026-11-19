#!/usr/bin/env bash
# CI local dos repositórios de serviço (RN_GIT_006, decisão do dono 2026-10-26): no GitHub só rodam os
# fluxos do baremetal e o cd-baremetal-staging.yml; os demais workflows estão desligados e rodam AQUI,
# em container, com o `act` — o próprio .github/workflows/*.yml, sem reescrever passo.
# Instalado pelo baremetal/scripts/ci-local-servico/instalar.sh; o hook pre-push chama este script e só
# deixa o push passar com todos os workflows de pull_request verdes para o HEAD.
#
#   scripts/ci-local/run.sh                  → todos os workflows de PR para o HEAD; grava .cache/ci-local/ok/<sha>
#   scripts/ci-local/run.sh --workflow ci.yml → um workflow (iteração; NÃO grava o ok)
#   scripts/ci-local/run.sh --status          → diz se o HEAD já passou
#   scripts/ci-local/run.sh --marca           → imprime a marca <!-- ci-local: <sha> … --> para o corpo do PR
#   scripts/ci-local/run.sh --publicar        → depois do push: status `ci-local` success no commit, a prova que o
#                                               gate do cd-baremetal-staging (platform-actions ≥ v0.3.3) aceita
#
# Fora por desenho (deploy, nuvem ou contexto do PR): cd-*, rollback*, terraform*, release*,
# dependabot-*, cleanup-*, pr-template*. Lista extra por repo em .ci-local-ignorar (um nome por linha).
set -euo pipefail
root=$(git rev-parse --show-toplevel); cd "$root"; here="$root/scripts/ci-local"
if [ -f "$root/.git" ]; then
  echo "CI local não roda em git worktree (o .git aponta para fora do container). Use um clone." >&2; exit 1
fi
# shellcheck source=/dev/null
. "$here/tools.lock"
so=""; [ "${1:-}" = --workflow ] && so="${2:?workflow}"
sha=$(git rev-parse HEAD); ok="$root/.cache/ci-local/ok/$sha"; logs="$root/.cache/ci-local/logs"
# Push que apaga referência (to_ref zerado) não leva código: passa. Outro commit que não o HEAD: recusa.
if [ -n "${PRE_COMMIT_TO_REF:-}" ] && [ -n "${PRE_COMMIT_TO_REF//0/}" ] && [ "$(git rev-parse "$PRE_COMMIT_TO_REF^{commit}" 2>/dev/null)" != "$sha" ]; then
  echo "pre-push de ${PRE_COMMIT_TO_REF:0:12} (não é o HEAD): faça checkout do commit e empurre de novo" >&2; exit 1
fi
mkdir -p "$(dirname "$ok")" "$logs"
case "${1:-}" in
  --status) if [ -f "$ok" ]; then echo "CI local OK para $sha"; exit 0; fi; echo "CI local AINDA NÃO passou para $sha"; exit 1;;
  --marca) [ -f "$ok" ] || { echo "CI local ainda não passou para $sha" >&2; exit 1; }
           echo "<!-- ci-local: $sha $(head -1 "$ok") -->"; exit 0;;
  --publicar) [ -f "$ok" ] || { echo "CI local ainda não passou para $sha" >&2; exit 1; }
           repo=$(gh repo view --json nameWithOwner -q .nameWithOwner)
           gh api "repos/$repo/commits/$sha" -q .sha >/dev/null 2>&1 \
             || { echo "$sha ainda não está no GitHub: faça o push antes de publicar" >&2; exit 1; }
           gh api -X POST "repos/$repo/statuses/$sha" -f state=success -f context=ci-local \
             -f description="$(sed -n 2p "$ok" | cut -c1-140)" -q '"status ci-local \(.state)"'; echo "em ${sha:0:12}"
           exit 0;;
esac
if [ -z "$so" ] && [ -f "$ok" ]; then echo "CI local já passou para $sha — nada a fazer"; exit 0; fi
if [ -z "$so" ] && [ -n "$(git status --porcelain --untracked-files=normal)" ]; then
  echo "árvore suja: commite (ou guarde) antes — o resultado vale para um commit exato" >&2; exit 1
fi
# Base do PR: CI_LOCAL_BASE (ex.: origin/develop) ou o branch padrão do remoto.
base_ref=${CI_LOCAL_BASE:-$(git symbolic-ref -q --short refs/remotes/origin/HEAD 2>/dev/null || echo origin/main)}
git fetch -q origin "+refs/heads/${base_ref#origin/}:refs/remotes/$base_ref"
# Base calculada ANTES da fila e do lock. Sem história comum (no exchangeos, main e develop são disjuntas) o
# merge-base sai 1 calado: sem CI_LOCAL_BASE, tenta a origin/develop; senão para aqui, dizendo o que fazer.
if ! base=$(git merge-base "$base_ref" HEAD 2>/dev/null); then
  if [ -z "${CI_LOCAL_BASE:-}" ] && [ "$base_ref" != origin/develop ] \
    && git fetch -q origin "+refs/heads/develop:refs/remotes/origin/develop" 2>/dev/null \
    && base=$(git merge-base origin/develop HEAD 2>/dev/null); then
    echo "$base_ref não tem história comum com o HEAD: base do PR = origin/develop"; base_ref=origin/develop
  else
    echo "sem merge-base entre $base_ref e o HEAD: rode com CI_LOCAL_BASE=<ref> (ex.: origin/develop)" >&2; exit 1
  fi
fi
if ! command -v docker >/dev/null || ! docker info >/dev/null 2>&1; then echo "Docker não está rodando" >&2; exit 1; fi

# Workflows que respondem a pull_request, menos os de deploy/nuvem/contexto de PR.
ignorar='^(cd-|rollback|terraform|release|dependabot-|cleanup-|pr-template)'
extra=$( [ -f .ci-local-ignorar ] && grep -vE '^\s*(#|$)' .ci-local-ignorar || true)
fluxos=()
for f in .github/workflows/*.y*ml; do
  n=$(basename "$f")
  [ -n "$so" ] && { [ "$n" = "$so" ] && fluxos+=("$f"); continue; }
  grep -qE "$ignorar" <<<"$n" && continue
  [ -n "$extra" ] && grep -qxF "$n" <<<"$extra" && continue
  grep -qE '^\s{2}pull_request:?\s*$|^on:.*\bpull_request\b|^\s+- pull_request\s*$' "$f" || continue
  fluxos+=("$f")
done
[ ${#fluxos[@]} -gt 0 ] || { echo "nenhum workflow de pull_request para rodar${so:+ ($so)}" >&2; exit 1; }

# Fila única do gate pesado (dono, 2026-10-27, O194), por ordem de chegada: espera a vez de TODO gate pesado da
# máquina (fila-pesado.sh), ANTES do lock do act, para não segurar os gates leves enquanto espera. Duas pistas
# (dono, 2026-11-19): o hook do exchangeos segurou uma vaga de BANCO das 23:11 às 09:31 só por `go test -race`,
# e pela ADR-039 o teste da MS é unitário, sem banco nem broker.
#   - pista banco: algum workflow incluído (ou um `uses: ./…` local dele) SOBE banco/broker, isto é, casa
#     banco_re (cockroach|crdb|postgres|kafka|redpanda|strimzi) DENTRO de um bloco `services:` (as linhas mais
#     indentadas que a chave), ou numa linha de `docker run` (com as continuações `\` juntas); ou usa
#     `docker compose`/`docker-compose`/testcontainers, qualquer que seja a imagem (a imagem fica fora do YAML:
#     na dúvida, banco). Sem pyyaml: os repos de serviço podem não ter.
#   - pista leve: nenhum sobe banco, mas algum casa pesado_re (`go test -race`, e2e sem banco…). Só o fila_entrar
#     vê REVENU_GATE_SEM_BANCO=1 (o act não herda); no modo comando da fila, o fila_confere_leve segue mandando
#     à pista com banco quem declara leve e sobe banco.
#   - nada casa: sem fila.
pesado_re='cockroach|crdb|kafka|redpanda|strimzi|e2e|go test[^#]*-race'
banco_re='cockroach|crdb|postgres|kafka|redpanda|strimzi'
sobe_banco() { # <arquivo>: 0 se o YAML sobe banco/broker pelo critério acima (awk POSIX, sem pyyaml)
  awk -v re="$banco_re" '
    function ind(s) { match(s, /^ */); return RLENGTH }
    BEGIN { sv = -1; acc = ""; achou = 0 }
    {
      l = tolower($0); sub(/\r$/, "", l)
      if (sv >= 0 && l !~ /^[ \t]*(#.*)?$/ && ind(l) <= sv) sv = -1   # fim do bloco services:
      if (sv >= 0 && l ~ re) { achou = 1; exit }
      if (sv < 0 && l ~ /^ *services: *(#.*)?$/) sv = ind(l)
      j = (acc == "") ? l : acc " " l
      if (j ~ /\\ *$/) { sub(/\\ *$/, "", j); acc = j; next }        # continuação: junta a linha seguinte
      acc = ""
      if (j ~ /docker[ -]compose|testcontainers/) { achou = 1; exit }
      if (j ~ /docker +(container +)?run/ && j ~ re) { achou = 1; exit }
    }
    END { exit !achou }' "$1"
}
locais() { # <workflow>: o próprio arquivo e os `uses: ./…` locais dele (workflow reutilizável ou action composta)
  local u a; echo "$1"
  { grep -oE 'uses:[[:space:]]*["'\'']?\./[^"'\''@[:space:]]+' "$1" || true; } \
    | sed -E 's/^uses:[[:space:]]*["'\'']?\.\///' \
    | while IFS= read -r u; do
        if [ -f "$u" ]; then echo "$u"; continue; fi
        for a in "$u/action.yml" "$u/action.yaml"; do if [ -f "$a" ]; then echo "$a"; fi; done
      done
}
# Runners com imagem no CI local (-P do act). Perna de matrix ou job cujo runner não está aqui (macos-*, windows-*,
# self-hosted…) fica FORA, com aviso, e nunca conta como rodada. No exchangeos (2026-11-19), o `test` com
# `runs-on: ${{ matrix.os }}` e os: [ubuntu-latest, macos-latest, windows-latest] saía às vezes com «Start image=»
# vazio e «invalid reference format» na perna ubuntu: o act 0.2.89 avalia o runs-on das pernas em paralelo.
#   - matrix: `--matrix <chave>:<valor>` só com os valores mapeados da chave usada no runs-on (o act filtra só as
#     combinações que TÊM a chave; job sem ela na matrix fica inteiro). Outro job com a mesma chave fora do runs-on
#     entra com os valores dele. Chave sem valor mapeado: um valor sentinela, que zera as pernas dela.
#   - runs-on literal sem imagem: o act já pula sem rodar («Skipping unsupported platform»); aqui só se avisa e
#     conta. Sem `-j`: ele roda um job só (com os needs dele) e quebraria os `needs` entre os demais.
#   - runs-on com outra expressão, ou matrix por expressão: fica com o act, sem filtro.
#   - workflow com todas as pernas fora: sem prova (rc≠0), para não gerar marca de CI vazio.
rotulos=(ubuntu-latest ubuntu-24.04 ubuntu-26.04)
pernas() { # <workflow>: M<TAB>chave:valor (--matrix), F<TAB>job<TAB>runner (perna fora), N<TAB>n (pernas que rodam)
  awk -v rot="${rotulos[*]}" -v q="'" '
    function ind(s) { match(s, /^ */); return RLENGTH }
    function limpa(s,  a, b) {
      sub(/^[ \t]+/, "", s); sub(/[ \t]+$/, "", s); a = substr(s, 1, 1); b = substr(s, length(s), 1)
      if (length(s) >= 2 && a == b && (a == "\"" || a == q)) s = substr(s, 2, length(s) - 2)
      return s
    }
    function poe(j, k, v) {
      v = limpa(v); if (v == "" || ((j, k, v) in visto)) return
      visto[j, k, v] = 1; tem[j, k] = 1; nv[j, k]++; vv[j, k, nv[j, k]] = v
    }
    function lista(s, j, k,  a, n, i) {   # [a, b, c] → poe de cada um
      sub(/^\[/, "", s); sub(/\][ \t]*$/, "", s); n = split(s, a, ",")
      for (i = 1; i <= n; i++) poe(j, k, a[i])
    }
    function roda_lista(s,  a, n, i, r) {   # [a, b] → a,b
      sub(/^\[/, "", s); sub(/\][ \t]*$/, "", s); n = split(s, a, ","); r = ""
      for (i = 1; i <= n; i++) r = (r == "" ? "" : r ",") limpa(a[i])
      return r
    }
    BEGIN { nj = 0; ji = -1; nos = 0; acc = "" }
    {
      l = $0; sub(/\r$/, "", l); sub(/[ \t]+#.*$/, "", l)
      if (l ~ /^[ \t]*(#.*)?$/) next
      n = ind(l); t = substr(l, n + 1)
      if (acc != "") {                                   # lista [ … ] em várias linhas
        acc = acc " " t; if (t !~ /\]/) next
        if (alvo == "ro") ro[job] = roda_lista(acc); else lista(acc, job, chave)
        acc = ""; next
      }
      if (n == 0) { nos = (t ~ /^jobs:[ \t]*$/); job = ""; next }
      if (!nos) next
      if (ji < 0) ji = n
      if (n <= ji) {
        job = ""
        if (n == ji && t ~ /^[A-Za-z0-9_-]+:[ \t]*$/) { job = t; sub(/:.*/, "", job); jobs[++nj] = job; jc = -1; ctx = "" }
        next
      }
      if (job == "") next
      if (jc < 0) jc = n
      if (n == jc && ctx == "ro" && t ~ /^- /) { v = limpa(substr(t, 3)); ro[job] = (ro[job] == "" ? v : ro[job] "," v); next }
      if (n <= jc) {                                     # chave do job
        ctx = ""
        if (t ~ /^runs-on:/) {
          v = t; sub(/^runs-on:[ \t]*/, "", v)
          if (v == "") { ctx = "ro"; ro[job] = "" }
          else if (v ~ /^\[/ && v !~ /\]/) { acc = v; alvo = "ro" }
          else if (v ~ /^\[/) ro[job] = roda_lista(v)
          else ro[job] = limpa(v)
        } else if (t ~ /^strategy:[ \t]*$/) ctx = "st"
        next
      }
      if (ctx == "ro") {                                 # runs-on em lista de bloco; group/labels: com o act
        if (t ~ /^- /) { v = limpa(substr(t, 3)); ro[job] = (ro[job] == "" ? v : ro[job] "," v) }
        else { ro[job] = "?"; ctx = "" }
        next
      }
      if (ctx == "mx" && n <= mi) ctx = "st"
      if (ctx == "st") {
        if (t ~ /^matrix:/) { v = t; sub(/^matrix:[ \t]*/, "", v); if (v != "") din[job] = 1; else { ctx = "mx"; mi = n; mk = -1 } }
        next
      }
      if (ctx != "mx") next
      if (mk < 0) mk = n
      if (n <= mk && t !~ /^- /) {                       # chave da matrix (o «- » no mesmo nível é item dela)
        chave = t; sub(/:.*/, "", chave); v = t; sub(/^[^:]*:[ \t]*/, "", v)
        sec = (chave == "include") ? "inc" : (chave == "exclude") ? "exc" : "val"
        if (sec == "val") {
          tem[job, chave] = 1
          if (v ~ /\$\{\{/) dk[job, chave] = 1
          else if (v ~ /^\[/ && v !~ /\]/) { acc = v; alvo = "mx" }
          else if (v ~ /^\[/) lista(v, job, chave)
        }
        next
      }
      if (sec == "val" && t ~ /^- /) poe(job, chave, substr(t, 3))
      if (sec == "inc") {                                # include: os valores da chave contam também
        s = t; sub(/^- +/, "", s)
        if (s ~ /^[A-Za-z0-9_-]+:[ \t]*[^ \t]/) { k2 = s; sub(/:.*/, "", k2); v = s; sub(/^[^:]*:[ \t]*/, "", v); poe(job, k2, v) }
      }
    }
    END {
      split(rot, ra, " "); for (i in ra) img[ra[i]] = 1
      roda = 0; fora = 0
      for (i = 1; i <= nj; i++) {
        j = jobs[i]; r = ro[j]
        if (r ~ /^\$\{\{ *matrix\.[A-Za-z0-9_-]+ *\}\}$/) {
          k = r; sub(/^\$\{\{ *matrix\./, "", k); sub(/ *\}\}$/, "", k)
          if ((j in din) || ((j, k) in dk) || !((j, k) in nv)) { roda++; continue }
          usa[k] = 1; noro[j, k] = 1
          for (x = 1; x <= nv[j, k]; x++) {
            v = vv[j, k, x]
            if (v in img) { pode[k, v] = 1; roda++ } else { print "F\t" j "\t" v; fora++ }
          }
        } else if (r == "" || r == "?" || r ~ /\$\{\{/) roda++
        else {
          m = split(r, lb, ","); achou = 0
          for (y = 1; y <= m; y++) if (lb[y] in img) achou = 1
          if (achou) roda++; else { print "F\t" j "\t" r; fora++ }
        }
      }
      for (x in tem) {                                   # mesma chave em job que não a usa no runs-on
        split(x, p, SUBSEP); j = p[1]; k = p[2]
        if (!(k in usa) || ((j, k) in noro)) continue
        if ((j in din) || ((j, k) in dk)) { sem[k] = 1; continue }
        for (y = 1; y <= nv[j, k]; y++) pode[k, vv[j, k, y]] = 1
      }
      for (k in usa) {
        if (k in sem) continue
        algum = 0
        for (x in pode) { split(x, p, SUBSEP); if (p[1] == k) { print "M\t" k ":" p[2]; algum = 1 } }
        if (!algum) print "M\t" k ":__ci-local-sem-imagem__"
      }
      print "N\t" roda
    }' "$1"
}
pesado=""; banco=""
for f in "${fluxos[@]}"; do
  eb=""; ep=""
  while IFS= read -r g; do
    [ -n "$eb" ] || ! sobe_banco "$g" || eb=1
    [ -n "$ep" ] || ! grep -qiE "$pesado_re" "$g" || ep=1
  done < <(locais "$f")
  [ -z "$eb" ] || banco="$banco $(basename "$f")"
  [ -z "$ep" ] || pesado="$pesado $(basename "$f")"
done
# shellcheck source=/dev/null
. "$here/fila-pesado.sh"
lock=${REVENU_CI_LOCK:-/tmp/revenu-ci-local.lock}; event=""
limpar() {
  [ -z "$event" ] || rm -f "$event"
  if [ "$(cat "$lock/pid" 2>/dev/null)" = "$$" ]; then rm -rf "$lock"; fi
  fila_sair
}
trap limpar EXIT
if [ -n "$banco" ]; then
  echo "gate pesado: pista banco —$banco"; REVENU_GATE_SEM_BANCO=0 fila_entrar "ci-local:$(basename "$root")"
elif [ -n "$pesado" ]; then
  echo "gate pesado: pista leve (sem banco) —$pesado"; REVENU_GATE_SEM_BANCO=1 fila_entrar "ci-local:$(basename "$root")"
fi

# UM act por máquina (mesmo lock do baremetal: porta 34567 e CPU emulada).
avisado=false
until mkdir "$lock" 2>/dev/null; do
  dono=$(cat "$lock/pid" 2>/dev/null || true)
  if [ -n "$dono" ] && ! kill -0 "$dono" 2>/dev/null; then rm -rf "$lock"; continue; fi
  [ "$avisado" = true ] || { echo "outro CI local rodando (PID ${dono:-?}, $(cat "$lock/quem" 2>/dev/null || echo '?')) — esperando…"; avisado=true; }
  sleep 15
done
echo "$$" > "$lock/pid"; echo "$root @ $(git rev-parse --short HEAD)" > "$lock/quem"

act=$("$here/install-act.sh")
imagem_local="revenu/ci-local-runner:${RUNNER_IMAGE##*sha256:}"; imagem_local="${imagem_local:0:40}"
if ! docker image inspect "$imagem_local" >/dev/null 2>&1; then
  echo "montando a imagem do runner local (uma vez): $imagem_local"
  # shellcheck disable=SC2016
  printf 'FROM %s\nRUN set -e; n=$(ls -d /opt/acttoolcache/node/*/x64/bin | sort -V | tail -1); for b in node npm npx; do ln -sf "$n/$b" /usr/local/bin/$b; done; node --version\n' "$RUNNER_IMAGE" \
    | docker build --platform linux/amd64 -q -t "$imagem_local" - >/dev/null
fi
# Token só pelo ambiente (em argv ele aparece no `ps`) e lido com o xtrace desligado: com `bash -x` a
# atribuição e qualquer teste sobre ele imprimem o valor no log. Daqui em diante, só pelo nome (-s GITHUB_TOKEN).
xt=""; case $- in *x*) xt=1;; esac; { set +x; } 2>/dev/null
if ! GITHUB_TOKEN=$(gh auth token 2>/dev/null) || [ -z "$GITHUB_TOKEN" ]; then
  echo "gh sem login: o act precisa do token para as ações privadas" >&2; exit 1
fi
export GITHUB_TOKEN
[ -z "$xt" ] || set -x
event=$(mktemp)
jq -n --arg b "$base" --arg h "$sha" --arg ref "$(git rev-parse --abbrev-ref HEAD)" --arg br "${base_ref#origin/}" \
  '{pull_request: {number: 0, base: {sha: $b, ref: $br}, head: {sha: $h, ref: $ref}, labels: []}}' > "$event"

plats=(); for r in "${rotulos[@]}"; do plats+=(-P "$r=$imagem_local"); done
start=$(date +%s); falhou=(); sem_prova=(); fora=()
for f in "${fluxos[@]}"; do
  n=$(basename "$f"); log="$logs/$sha-$n.log"
  matriz=(); fora_wf=(); roda=0; : > "$log"
  while IFS=$'\t' read -r tipo a b; do
    case $tipo in
      M) matriz+=(--matrix "$a");;
      F) fora_wf+=("$a/$b")
         echo "perna $a ($b) fora do CI local: sem imagem para $b" | tee -a "$log";;
      N) roda=$a;;
    esac
  done < <(pernas "$f")
  if [ ${#fora_wf[@]} -gt 0 ]; then
    fora+=("${fora_wf[@]}")
    if [ "$roda" -eq 0 ]; then
      echo "❌ $n: todas as pernas ficaram fora do CI local (${fora_wf[*]}): sem prova" | tee -a "$log"
      sem_prova+=("$n"); continue
    fi
  fi
  echo "▶ $n — log em ${log#"$root"/}"
  set +e
  "$act" pull_request -W "$f" -e "$event" "${plats[@]}" ${matriz[@]+"${matriz[@]}"} \
    --container-architecture linux/amd64 --pull=false --var LOCAL_CI=true \
    -s GITHUB_TOKEN --privileged --rm --concurrent-jobs 2 \
    --container-options "--cpus=${CI_LOCAL_CPUS:-2} --memory=${CI_LOCAL_MEMORIA:-3g}" \
    --artifact-server-path "$root/.cache/ci-local/artifacts" 2>&1 | tee -a "$log" | grep -E --line-buffered '🏁|::error'
  rc=${PIPESTATUS[0]}
  set -e
  if [ "$rc" -ne 0 ] || grep -q 'Job failed' "$log"; then falhou+=("$n"); fi
done
dur=$(( $(date +%s) - start ))
nota=""
if [ ${#fora[@]} -gt 0 ]; then
  pl=perna; if [ ${#fora[@]} -gt 1 ]; then pl=pernas; fi
  nota=$(printf '%s, ' "${fora[@]}"); nota="${#fora[@]} $pl fora: ${nota%, }"
fi
if [ ${#sem_prova[@]} -gt 0 ]; then
  echo "❌ CI local SEM PROVA em ${dur}s: ${sem_prova[*]} sem nenhuma perna com imagem no CI local ($nota)" \
    "— workflow sem perna Linux não vira prova vazia: leve o caso ao dono"
  exit 1
fi
if [ ${#falhou[@]} -gt 0 ]; then
  echo "❌ CI local FALHOU em ${dur}s: ${falhou[*]} — corrija antes do push (logs em .cache/ci-local/logs/)"; exit 1
fi
if [ -z "$so" ]; then
  { date -u +%FT%TZ; echo "duração=${dur}s act=$ACT_VERSION workflows=${#fluxos[@]}${nota:+ ($nota)}"; } > "$ok"
fi
echo "✅ CI local verde em ${dur}s para $(git rev-parse --short HEAD) (${#fluxos[@]} workflows${nota:+; $nota})"
