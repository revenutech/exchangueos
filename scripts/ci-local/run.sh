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
#   scripts/ci-local/run.sh --sem-cache [...] → a rodada do push à mão: go test sem cache de resultado (abaixo)
#
# Rodada do push sem cache de teste (dono, «Aplica no push», 23/11/2026, DV 2930; verde falso por
# cache no ledgeros, DV 2927): no hook pre-push (PRE_COMMIT_TO_REF no ambiente, que o pre-commit
# exporta no estágio pre-push e os blocos de husky e lefthook do instalar.sh passam) ou com
# --sem-cache, o `go test` roda com -count=1: no padrão da N1 unitária, na linha de comando; nos
# passos do .ci-local-modo-n1, por GOFLAGS (-count=1 acrescentado ao que já houver; o go só o aplica
# ao `go test`); no act, por `--env GOFLAGS=-count=1` (o GOCACHE volta do cache do actions/cache
# via ~/.cache/actcache; env do workflow ou do passo prevalece). À mão, sem a flag, segue com
# cache. A prova gravada à mão com cache não libera o push: o hook roda de novo, sem cache. A nota
# vai à 3ª linha da prova, fora da descrição do status (o gate do CD casa a descrição inteira).
#
# Modo unitário (dono, DV 2397, 22/11/2026: enquanto o plano do módulo não fecha, só a N1
# unitária): a linha «unitario» no .ci-local-modo da raiz do repo troca o act por
# a N1 declarada no .ci-local-modo-n1 (um passo por linha) ou, sem ela, `go test -short` no
# go.mod da raiz; sem banco, act nem mutante, na pista leve da fila; verde, publica o
# status `gate-local` no commit (ver n1_unitaria).
#
# Fora por desenho (deploy, nuvem ou contexto do PR): cd-*, rollback*, terraform*, release*,
# dependabot-*, cleanup-*, pr-template*. Lista extra por repo em .ci-local-ignorar (um nome por linha).
set -euo pipefail
sem_cache=""; if [ "${1:-}" = --sem-cache ]; then sem_cache=1; shift; fi
root=$(git rev-parse --show-toplevel); cd "$root"; here="$root/scripts/ci-local"
# Modo do pre-push: a 1ª linha útil (# comenta) do .ci-local-modo; sem o arquivo, o de sempre
# (act). Lido antes da checagem de worktree: o unitário não sobe contêiner e roda em worktree
# (dono, 22/11/2026; toda sessão empurra de worktree); o act (completo ou --workflow), não.
modo=""
if [ -f .ci-local-modo ]; then
  modo=$(grep -vE '^[[:space:]]*(#|$)' .ci-local-modo | head -1 | tr -d '[:space:]' || true)
fi
case "$modo" in
  ''|completo) modo="";;
  unitario) ;;
  *) echo ".ci-local-modo: modo «${modo}» desconhecido (completo ou unitario)" >&2; exit 1;;
esac
if [ -f "$root/.git" ] && { [ "$modo" != unitario ] || [ "${1:-}" = --workflow ]; }; then
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
# Rodada que libera o push (DV 2930): o hook, ou à mão com --sem-cache. A prova dela leva a nota na
# 3ª linha (a 2ª é a descrição do status, que o gate do CD casa inteira).
[ -z "${PRE_COMMIT_TO_REF:-}" ] || sem_cache=1
nota_sem_cache="sem cache (DV 2930)"
ok_vale() { # 0 se há prova do HEAD que serve a esta rodada (na do push, só a gravada sem cache)
  [ -f "$ok" ] || return 1
  [ -z "$sem_cache" ] && return 0
  grep -qxF "$nota_sem_cache" "$ok" && return 0
  echo "a prova de ${sha:0:12} foi com cache de teste: a rodada do push roda de novo, sem cache" \
    "(DV 2930)"
  return 1
}
# A prova do modo unitário fica em ok-unitario/ e vai ao status `gate-local`: nunca vale como
# CI completo (ci-local).
contexto=ci-local
if [ "$modo" = unitario ]; then ok="$root/.cache/ci-local/ok-unitario/$sha"; contexto=gate-local; fi
mkdir -p "$(dirname "$ok")" "$logs"
repo_origin() { # owner/repo do remoto origin (ssh ou https); vazio se não der. Nunca o `gh repo
  # view`: num fork ele resolve o repo pai (FlinkOS, 22/11: apache/flink, status recusado). Lê a URL
  # como está na config (sem o insteadOf), a mesma do `git remote get-url` sem reescrita
  git config --get remote.origin.url 2>/dev/null | sed -nE \
    's#^(git@[^:/]+:|ssh://([^@/]+@)?[^/]+/|https?://([^@/]+@)?[^/]+/)([^/]+/[^/]+)$#\4#p' \
    | sed -E 's#/$##; s#\.git$##' | grep -E '^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$' || true
}
publicar() { # status $contexto success no $sha (descrição: 2ª linha do $ok); 1 se não deu.
  # Diz o repo alvo e, na falha, o rc e a mensagem curta do gh (vão ao log do retry)
  local repo e rc=0
  repo=$(repo_origin)
  if [ -z "$repo" ]; then
    echo "publicar: não extraí owner/repo do origin «$(git config --get remote.origin.url \
      | sed -E 's#//[^@/]+@#//#')»; recusado (sem cair no gh repo view)" >&2; return 1
  fi
  echo "publicar: alvo $repo, ${sha:0:12}, status $contexto" >&2
  # sha fora do origin (push que não entrou: o SSH caiu no hook, KeycloakOS 22/11): a API de statuses
  # responde 422 «No commit found for SHA»; diz isso no log, sem publicar por outro caminho
  e=$(gh api "repos/$repo/commits/$sha" -q .sha 2>&1) || rc=$?
  if [ "$rc" != 0 ]; then
    echo "publicar: sha ainda não está no origin ($repo, ${sha:0:12}; gh rc $rc): ${e:0:160}" >&2; return 1
  fi
  e=$(gh api -X POST "repos/$repo/statuses/$sha" -f state=success -f context="$contexto" \
    -f description="$(sed -n 2p "$ok" | cut -c1-140)" \
    -q '"status \(.context) \(.state)"' 2>&1) || rc=$?
  if [ "$rc" != 0 ]; then
    case "$e" in *"No commit found"*) e="sha ainda não está no origin: $e" ;; esac
    echo "publicar: POST do status em $repo (gh rc $rc): ${e:0:160}" >&2; return 1
  fi
  echo "$e em $repo@${sha:0:12}"
}
java_do_repo() { # JDK do repo para os passos (FlinkOS, IcebergOS e KeycloakOS, 22/11: o hook herdava o
  # OpenJDK 26 do Mac, e eles compilam em 21). Ordem: .ci-local-java, .java-version, java= do
  # .sdkmanrc. Sem arquivo, não mexe; JDK pedido e ausente: falha dizendo a versão e o arquivo
  local f v="" jh
  for f in .ci-local-java .java-version .sdkmanrc; do
    [ -f "$f" ] || continue
    if [ "$f" = .sdkmanrc ]; then v=$(grep -E '^[[:space:]]*java=' "$f" | head -1 | cut -d= -f2- || true)
    else v=$(grep -vE '^[[:space:]]*(#|$)' "$f" | head -1 || true); fi
    v=$(printf '%s' "$v" | tr -d '[:space:]' | sed -E 's/^[^0-9]*//; s/^([0-9]+(\.[0-9]+)*).*/\1/')
    [ -z "$v" ] || break
  done
  [ -n "$v" ] || return 0
  jh=$(command -v java_home 2>/dev/null || echo /usr/libexec/java_home)
  if ! JAVA_HOME=$("$jh" -v "$v" 2>/dev/null) || [ -z "$JAVA_HOME" ]; then
    echo "JDK $v (pedido em $f) não está instalado nesta máquina: instale-o (ex.: brew install" \
      "--cask temurin@${v%%.*}) e rode de novo" >&2; exit 1
  fi
  export JAVA_HOME PATH="$JAVA_HOME/bin:$PATH"
  echo "JDK $v (de $f): JAVA_HOME=$JAVA_HOME"
}
pacotes_pact() { # diretórios com _test.go de contrato pact (//go:build com pact ou import do pact-go)
  git grep -lE '^//go:build.*(^|[^[:alnum:]_])pact([^[:alnum:]_]|$)|"github\.com/pact-foundation/pact-go' \
    -- '*_test.go' 2>/dev/null | xargs -n1 dirname 2>/dev/null | sort -u || true
}
lembra_push() { # no hook: o «Passed» não prova que o push entrou (KeycloakOS, 22/11: o SSH caiu depois)
  [ -n "${PRE_COMMIT_TO_REF:-}" ] || return 0
  echo "confira que o push entrou: git ls-remote origin $(git rev-parse --abbrev-ref HEAD) deve" \
    "mostrar ${sha:0:12}"
}
hook_sem_vaga() { # pista leve, sem prova: no hook pre-push, sai já se a fila não tem vaga agora
  # (dono, 2026-11-22: o GitHub derruba o SSH do push que espera vaga). À mão, espera como sempre.
  local r
  if [ -z "${PRE_COMMIT_TO_REF:-}" ] || ! declare -F fila_tem_vaga >/dev/null; then return 0; fi
  if r=$( (fila_tem_vaga leve) 2>&1); then return 0; fi
  printf '%s\n' "$r" "fila sem vaga: rode \`scripts/ci-local/run.sh --sem-cache\` (sem push) e" \
    "empurre de novo; a marca sem cache faz o hook passar na hora (DV 2930)" >&2
  exit 75
}
case "${1:-}" in
  --status) if [ -f "$ok" ]; then echo "CI local OK para $sha${modo:+ (modo $modo)}"; exit 0; fi
            echo "CI local AINDA NÃO passou para $sha${modo:+ (modo $modo)}"; exit 1;;
  --marca) if [ -n "$modo" ]; then
             echo "modo $modo: a prova é o status $contexto (--publicar), não a marca" >&2; exit 1
           fi
           [ -f "$ok" ] || { echo "CI local ainda não passou para $sha" >&2; exit 1; }
           echo "<!-- ci-local: $sha $(head -1 "$ok") -->"; exit 0;;
  --publicar) [ -f "$ok" ] || { echo "CI local ainda não passou para $sha" >&2; exit 1; }
           publicar || { echo "$sha ainda não está no GitHub: faça o push antes" >&2; exit 1; }
           exit 0;;
esac
# Modo unitário (DV 2397): só a N1 unitária, sem as variáveis de banco e de broker (como o nível
# unitário do rodada-escopo.sh). Sem act, sem Docker, sem base do PR. A N1 (dono, 22/11/2026):
#   - declarada: cada linha útil (# comenta) do .ci-local-modo-n1 versionado é um passo, rodado
#     com `bash -c` na raiz: `[docker:] <comando shell>` ou `[docker:] passo <nome>` (o comando do
#     passo de disco <nome> do .ci-local/passos.conf, sem marcador n2/fresco/banco/integration/
#     e2e). Conta pass/fail pelo rc de cada um; verde = todos rc 0. A marca `docker:` deixa o passo
#     usar contêiner efêmero (sem banco nem broker) e põe a rodada na fila com REVENU_GATE_DOCKER=1;
#     sem ela, o passo vê um `docker` que recusa e um DOCKER_HOST inválido. Banco, broker, act e
#     mutante são recusados no texto do passo, e o `act` recusa em qualquer passo;
#   - sem declaração: `go test -short -json ./...` no go.mod DA RAIZ (go.mod só em subpasta, de
#     e2e ou tools, não testa o produto e não conta); verde = algum teste passou, nenhum falhou
#     e o go test saiu 0;
#   - nem uma nem outra: recusa o push (nunca verde vazio).
# Entra na pista leve da fila, sem REVENU_GATE_PESADA (e sem REVENU_GATE_DOCKER, salvo passo com
# a marca `docker:`). Verde: grava o $ok
# e publica o `gate-local`. O
# commit só chega ao GitHub depois do hook: se ainda não está lá, um processo à parte tenta
# CI_LOCAL_PUBLICAR_TENTATIVAS vezes (60), a cada CI_LOCAL_PUBLICAR_ESPERA_S (10 s); o gate do
# CD espera a prova por 30 min.
sem_env='(TEST_)?(DB|DATABASE)_(URL|DSN)|_IT_DATABASE_URL|TEST_DSN|COCKROACH|CRDB|KAFKA'
sem_env="$sem_env|REDPANDA|BROKER|BOOTSTRAP_SERVERS|SCHEMA_REGISTRY"
sem_env="$sem_env|^PG(HOST|HOSTADDR|PORT|USER|PASSWORD|DATABASE|SERVICE|SSLMODE)\$"
# Texto do passo da N1 (2ª camada; a 1ª é o runtime: sem `docker:`, o docker recusa). Act, mutante,
# compose e testcontainers casam no texto inteiro. Banco e broker casam só como ferramenta ou imagem:
# o n1_sem_caminhos tira os caminhos (FlinkOS, 22/11: `cd revenu/connectors/kafka` não é broker).
# Atribuição no texto a uma variável do $sem_env (KAFKA=…, DATABASE_URL=…) também recusa.
n1_proibido='(^|[^[:alnum:]_.-])act([[:space:]]|$)|mutac|mutant|mutation|gremlins|mutesting'
n1_proibido="$n1_proibido|docker[ -]compose|testcontainers"
n1_ferramenta='(^|[^[:alnum:]_.-])(kafka-(server-start|topics|console-[a-z]+|storage|configs'
n1_ferramenta="$n1_ferramenta|consumer-groups|run-class)|cockroach|postgres|pg_ctl|redpanda|rpk)"
n1_ferramenta="$n1_ferramenta([^[:alnum:]_]|\$)|confluentinc/|apache/kafka|bitnami/kafka"
n1_ferramenta="$n1_ferramenta|redpandadata/|cockroachdb/|postgres:|strimzi"
n1_sem_caminhos() { # <texto>: token com / sai, salvo imagem (depois de docker run|pull|create até o
  # fim do comando, ou depois de image:/FROM), URL, atribuição e comando chamado por caminho (fica
  # o nome do arquivo)
  printf '%s\n' "$1" | awk '{
    cmd = 1; img = 0; dk = 0; o = ""
    for (i = 1; i <= NF; i++) {
      t = $i; sep = (t ~ /^(&&|\|\||\||;)$/ || t ~ /;$/)
      lit = (t ~ /:\/\// || t ~ /^[A-Za-z_][A-Za-z0-9_]*=/)  # URL e atribuição ficam inteiras
      if (t ~ /\// && !lit && !img && dk < 2) { if (!cmd) t = ""; else { n = split(t, p, "/"); t = p[n] } }
      o = o " " t
      if (sep) { cmd = 1; img = 0; dk = 0; continue }
      img = (tolower($i) ~ /^(image:|from)$/)
      if (dk == 1) dk = ($i ~ /^(run|pull|create)$/) ? 2 : 0
      else if (dk == 0 && cmd && $i ~ /(^|\/)docker$/) dk = 1
      cmd = (t ~ /^[A-Za-z_][A-Za-z0-9_]*=/ || t ~ /^(sudo|env|exec|time|nice|sh|bash|xargs)$/ \
        || (cmd && t == "-c"))   # sh -c <comando>: o seguinte ainda é comando
    }
    print o }'
}
passo_conf() { # <nome>: «<marcadores>\t<comando>» do passo de disco <nome> no .ci-local/passos.conf
  awk -v p="$1" '!/^[[:space:]]*#/ && index($0, " -- ") {
    i0 = index($0, " -- "); n = split(substr($0, 1, i0 - 1), t, " "); m = ""; i = 1
    while (i < n && t[i] ~ /^(n2|fresco|banco|integration|e2e)$/) { m = m t[i] " "; i++ }
    if (t[i] == p) { print m "\t" substr($0, i0 + 4); exit }
  }' .ci-local/passos.conf 2>/dev/null
}
n1_plano() { # lê o .ci-local-modo-n1: N1_CMD[i], N1_DOCKER[i] (0|1); sai 1 com a linha errada
  local ln c cs d r m nl=0 tab=$'\t'
  N1_CMD=(); N1_DOCKER=()
  while IFS= read -r ln || [ -n "$ln" ]; do
    nl=$((nl + 1)); c=$(printf '%s' "$ln" | sed -E 's/^[[:space:]]+//; s/[[:space:]]+$//')
    case "$c" in ''|'#'*) continue;; esac
    d=0; case "$c" in 'docker:'*) d=1; c=${c#docker:}; c=${c#"${c%%[! ]*}"};; esac
    case "$c" in
      'passo '*) r=$(passo_conf "${c#passo }")
        if [ -z "$r" ]; then
          echo ".ci-local-modo-n1:$nl: passo «${c#passo }» não é passo de disco do" \
            ".ci-local/passos.conf (linha go: declare o comando direto)" >&2; exit 1
        fi
        m=${r%%"$tab"*}
        if [ -n "$m" ]; then
          echo ".ci-local-modo-n1:$nl: passo «${c#passo }» tem marcador ${m}e fica fora" \
            "da N1 unitária" >&2; exit 1
        fi
        c=${r#*"$tab"};;
    esac
    # aspas viram espaço: em `sh -c 'docker run …'` o token seria «'docker» (FlinkOS, 22/11)
    cs=$(printf '%s' "$c" | tr "\"'" '  ')
    if printf '%s\n' "$cs" | grep -qiE -- "$n1_proibido" \
      || n1_sem_caminhos "$cs" | grep -qiE -- "$n1_ferramenta" \
      || printf '%s\n' "$cs" | grep -oE '(^|[[:space:]])[A-Za-z_][A-Za-z0-9_]*=' | tr -d '[:space:]=' \
        | grep -qE -- "$sem_env"; then
      echo ".ci-local-modo-n1:$nl: banco, broker, act ou mutante não entram na N1 unitária" \
        "(DV 2397): $c" >&2; exit 1
    fi
    N1_CMD+=("$c"); N1_DOCKER+=("$d")
  done < .ci-local-modo-n1
  if [ ${#N1_CMD[@]} = 0 ]; then
    echo "modo unitário: .ci-local-modo-n1 sem nenhum passo (um por linha; # comenta)" >&2; exit 1
  fi
}
conta() { # <Action> <json>: testes (e subtestes) com a ação no -json do go test
  jq -Rn --arg a "$1" '[inputs | fromjson? | select(.Test != null and .Action == $a)] | length' "$2"
}
n1_unitaria() {
  local decl=0 docker=0 i n j e t0 dur p f k rc=0 v sd pact limpa=(env)
  if [ -f .ci-local-modo-n1 ]; then
    decl=1; n1_plano
    for i in "${N1_DOCKER[@]}"; do [ "$i" = 0 ] || docker=1; done
  elif ! git ls-files --error-unmatch go.mod >/dev/null 2>&1; then
    echo "modo unitário: sem .ci-local-modo-n1 e sem go.mod na raiz (go.mod em subpasta não" \
      "conta); push recusado. Crie e commite $root/.ci-local-modo-n1 com um passo de N1" \
      "por linha (# comenta): um comando shell (rodado na raiz), «passo <nome>» do" \
      ".ci-local/passos.conf ou «docker: …» para contêiner efêmero. Por exemplo:" >&2
    git ls-files -- '*/go.mod' | grep -vE '(^|/)(vendor|testdata)/' | sed -E \
      's|^(.*)/go.mod$|  go -C \1 test -short ./...|' >&2 || true
    echo "  <validador ou teste do produto, ex.: npx vitest run --maxWorkers=2>" >&2
    echo "ou tire o .ci-local-modo (volta ao CI completo)." >&2
    exit 1
  elif pact=$(pacotes_pact) && [ -n "$pact" ]; then
    # contrato pact não linka sem a pact_ffi (AuthorityOS, 22/11): o `go test ./...` reprovaria. A fila
    # nunca tira pacote por conta própria (afrouxaria o gate): quem declara o recorte é o repo
    echo "modo unitário: sem .ci-local-modo-n1 e com testes de contrato pact (precisam da pact_ffi)" \
      "em: $(printf '%s' "$pact" | tr '\n' ' ')— push recusado antes de rodar. Declare a N1 em" \
      "$root/.ci-local-modo-n1 (um passo por linha; por exemplo, o go test dos pacotes sem" \
      "pact, ou «passo <nome>» do .ci-local/passos.conf) e commite." >&2
    exit 1
  fi
  for v in $(compgen -e | grep -E -- "$sem_env" | grep -Ev '^(REVENU_GATE|RODADA)_' || true); do
    limpa+=(-u "$v")
  done
  # Os passos rodam sem os GIT_* que o hook herda (OnboardOS, 22/11: com o GIT_DIR do pre-push, o
  # `git init` em tempdir dos testes lia o repo real, 13485/2 contra 13488/0 no clone). O run.sh
  # já os usou; só os passos rodam limpos. Lista do próprio git (--local-env-vars) e a do hook.
  for v in $(git rev-parse --local-env-vars 2>/dev/null) GIT_DIR GIT_WORK_TREE GIT_INDEX_FILE \
    GIT_COMMON_DIR GIT_OBJECT_DIRECTORY GIT_ALTERNATE_OBJECT_DIRECTORIES GIT_PREFIX \
    GIT_QUARANTINE_PATH $(compgen -e | grep -E '^GIT_(PUSH_OPTION_|CONFIG_(KEY|VALUE)_)' || true); do
    limpa+=(-u "$v")
  done
  # Ferramentas Go (golangci-lint & cia.) moram no $(go env GOPATH)/bin, fora do PATH do hook
  # (SocketOS, 22/11: rc 127): entra no fim do PATH, se houver go e o diretório existir.
  if command -v go >/dev/null 2>&1; then
    v=$(go env GOPATH 2>/dev/null || true); v=${v%%:*}
    if [ -n "$v" ] && [ -d "$v/bin" ]; then
      case ":$PATH:" in *":$v/bin:"*) ;; *) PATH="$PATH:$v/bin";; esac
    fi
  fi
  # Rodada do push (DV 2930): go test sem cache de resultado. Nos passos, pelo GOFLAGS (o go só
  # aplica o -count ao `go test`); no padrão, também na linha de comando (abaixo).
  if [ -n "$sem_cache" ]; then limpa+=("GOFLAGS=${GOFLAGS:+$GOFLAGS }-count=1"); fi
  java_do_repo   # antes da senha: JDK ausente falha sem pegar a fila
  # shellcheck source=/dev/null
  . "$here/fila-pesado.sh"
  trap fila_sair EXIT
  unset REVENU_GATE_DOCKER REVENU_GATE_PESADA
  REVENU_GATE_DOCKER=$docker hook_sem_vaga
  if [ "$docker" = 1 ]; then
    echo "N1 unitária (DV 2397): pista leve da fila, com passo docker (sem banco, act nem mutante)"
    REVENU_GATE_DOCKER=1 REVENU_GATE_SEM_BANCO=1 REVENU_GATE_NIVEL_TESTE=unitaria \
      REVENU_GATE_NIVEL_PROVA=N1 fila_entrar "ci-local-n1:$(basename "$root")"
  else
    echo "N1 unitária (DV 2397): pista leve da fila, sem act nem contêiner"
    REVENU_GATE_SEM_BANCO=1 REVENU_GATE_NIVEL_TESTE=unitaria REVENU_GATE_NIVEL_PROVA=N1 \
      fila_entrar "ci-local-n1:$(basename "$root")"
  fi
  j=$logs/$sha-n1-unitaria.json; e=$logs/$sha-n1-unitaria.err; : > "$j"; : > "$e"
  t0=$(date +%s)
  if [ "$decl" = 1 ]; then
    # docker e act de recusa: act em todo passo; docker (e DOCKER_HOST inválido, para quem fala
    # com a API direto) só no passo sem a marca
    sd="$root/.cache/ci-local/n1-sem-docker"; mkdir -p "$sd/act" "$sd/docker"
    printf '#!/bin/sh\necho "N1 unitária: act proibido (DV 2397)" >&2; exit 97\n' > "$sd/act/act"
    printf '#!/bin/sh\necho "N1 unitária: passo sem a marca docker chamou o docker" >&2\n%s\n' \
      'exit 97' > "$sd/docker/docker"
    chmod +x "$sd/act/act" "$sd/docker/docker"
    n=${#N1_CMD[@]}; p=0; f=0
    for i in $(seq 0 $((n - 1))); do
      if [ "${N1_DOCKER[$i]}" = 1 ]; then
        echo "▶ N1 declarada $((i + 1)) (docker): ${N1_CMD[$i]}"; set -- PATH="$sd/act:$PATH"
      else
        echo "▶ N1 declarada $((i + 1)): ${N1_CMD[$i]}"
        set -- PATH="$sd/act:$sd/docker:$PATH" DOCKER_HOST=unix:///n1-unitaria/sem-docker
      fi
      if "${limpa[@]}" "$@" bash -c "${N1_CMD[$i]}" < /dev/null >> "$e" 2>&1; then p=$((p + 1))
      else f=$((f + 1)); echo "  FAIL passo $((i + 1)): ${N1_CMD[$i]}"; fi
    done
    dur=$(( $(date +%s) - t0 ))
    if [ "$f" != 0 ] || [ "$p" = 0 ]; then
      tail -30 "$e"
      echo "❌ N1 unitária FALHOU em ${dur}s: $n comandos, $p pass/$f fail —" \
        "sem gate-local; corrija antes do push (log em ${e#"$root"/})"
      exit 1
    fi
    printf '%s\nN1 unitária (2397): %s comandos, %s pass/%s fail, %ss\n%s' \
      "$(date -u +%FT%TZ)" "$n" "$p" "$f" "$dur" "${sem_cache:+$nota_sem_cache$'\n'}" > "$ok"
    echo "✅ N1 unitária verde em ${dur}s para ${sha:0:12}: $n comandos, $p pass/$f fail"
    return 0
  fi
  if [ -n "$sem_cache" ]; then
    echo "▶ go test -short -count=1 ./... na raiz (rodada do push, sem cache; DV 2930)"
    "${limpa[@]}" go test -short -count=1 -json ./... >> "$j" 2>> "$e" || rc=1
  else
    echo "▶ go test -short ./... na raiz"
    "${limpa[@]}" go test -short -json ./... >> "$j" 2>> "$e" || rc=1
  fi
  dur=$(( $(date +%s) - t0 )); p=$(conta pass "$j"); f=$(conta fail "$j"); k=$(conta skip "$j")
  if [ "$rc" != 0 ] || [ "$f" != 0 ] || [ "$p" = 0 ]; then
    jq -Rr 'fromjson? | select(.Action == "fail") | "  FAIL \(.Package) \(.Test // "(pacote)")"' \
      "$j" | head -40
    tail -20 "$e"
    echo "❌ N1 unitária FALHOU em ${dur}s: $p pass/$f fail, $k skip, go test rc=$rc —" \
      "sem gate-local; corrija antes do push (log em ${j#"$root"/})"
    exit 1
  fi
  printf '%s\nN1 unitária (2397): go test -short, %s pass/%s fail, %s skip, %ss\n%s' \
    "$(date -u +%FT%TZ)" "$p" "$f" "$k" "$dur" "${sem_cache:+$nota_sem_cache$'\n'}" > "$ok"
  echo "✅ N1 unitária verde em ${dur}s para ${sha:0:12}: $p pass/$f fail, $k skip"
}
if [ "$modo" = unitario ] && [ -z "$so" ]; then
  if ok_vale; then echo "N1 unitária já passou para $sha"
  else
    if [ -n "$(git status --porcelain --untracked-files=normal)" ]; then
      echo "árvore suja: commite (ou guarde) antes — o resultado vale para um commit exato" >&2
      exit 1
    fi
    n1_unitaria
  fi
  if publicar; then lembra_push; exit 0; fi
  pl="$logs/$sha-gate-local.log"
  echo "gate-local: ${sha:0:12} ainda não está no GitHub; publico à parte depois do push" \
    "(log em ${pl#"$root"/})"
  ( trap - EXIT
    for i in $(seq 1 "${CI_LOCAL_PUBLICAR_TENTATIVAS:-60}"); do
      sleep "${CI_LOCAL_PUBLICAR_ESPERA_S:-10}"; if publicar; then exit 0; fi
      echo "$(date -u +%FT%TZ) tentativa $i falhou"
    done
    echo "desisti: ${sha:0:12} não apareceu no GitHub; rode scripts/ci-local/run.sh --publicar"
  ) < /dev/null >> "$pl" 2>&1 &
  disown || true
  lembra_push; exit 0
fi
if [ -z "$so" ] && ok_vale; then echo "CI local já passou para $sha — nada a fazer"; lembra_push; exit 0; fi
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
  # o act sobe contêiner (hadolint, dockle e os jobs): espera a memória da VM do Docker (dono, 2026-11-22)
  REVENU_GATE_DOCKER=1 hook_sem_vaga; echo "gate pesado: pista leve (sem banco) —$pesado"
  REVENU_GATE_SEM_BANCO=1 REVENU_GATE_DOCKER=1 fila_entrar "ci-local:$(basename "$root")"
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
# Rodada do push (DV 2930): o setup-go devolve o GOCACHE (com os resultados de teste) pelo cache do
# act em ~/.cache/actcache; o -count=1 no GOFLAGS desliga o cache de teste. O env do workflow ou do
# passo que define GOFLAGS prevalece (cardos e investos zeram o GOFLAGS num passo).
cache_env=(); if [ -n "$sem_cache" ]; then cache_env=(--env GOFLAGS=-count=1); fi
start=$(date +%s); falhou=(); sem_prova=(); fora=()
# Teto de CPU por job (2026-10-26, malha `revenu` caiu com a VM do Docker a 2487 %): 2 jobs × 2 CPUs. Vale só para o
# container do job; os `services:` (CRDB, Kafka) o act cria sem essa opção. CI_LOCAL_CPUS muda o teto.
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
    ${cache_env[@]+"${cache_env[@]}"} \
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
  { date -u +%FT%TZ; echo "duração=${dur}s act=$ACT_VERSION workflows=${#fluxos[@]}${nota:+ ($nota)}"
    if [ -n "$sem_cache" ]; then echo "$nota_sem_cache"; fi; } > "$ok"
fi
echo "✅ CI local verde em ${dur}s para $(git rev-parse --short HEAD) (${#fluxos[@]} workflows${nota:+; $nota})"
lembra_push
