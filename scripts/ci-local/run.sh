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

# Fila única do gate pesado (dono, 2026-10-27, O194), por ordem de chegada: workflow com CRDB, Kafka, e2e ou
# `go test -race` espera a vez de TODO gate pesado da máquina (fila-pesado.sh), ANTES do lock do act, para não
# segurar os gates leves enquanto espera.
pesado_re='cockroach|crdb|kafka|redpanda|strimzi|e2e|go test[^#]*-race'
pesado=""; for f in "${fluxos[@]}"; do grep -qiE "$pesado_re" "$f" && pesado="$pesado $(basename "$f")"; done
# shellcheck source=/dev/null
. "$here/fila-pesado.sh"
lock=${REVENU_CI_LOCK:-/tmp/revenu-ci-local.lock}; event=""
limpar() {
  [ -z "$event" ] || rm -f "$event"
  if [ "$(cat "$lock/pid" 2>/dev/null)" = "$$" ]; then rm -rf "$lock"; fi
  fila_sair
}
trap limpar EXIT
if [ -n "$pesado" ]; then echo "gate pesado:$pesado"; fila_entrar "ci-local:$(basename "$root")"; fi

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
# Token só pelo ambiente (em argv ele aparece no `ps`).
GITHUB_TOKEN=$(gh auth token 2>/dev/null) || { echo "gh sem login: o act precisa do token para as ações privadas" >&2; exit 1; }
export GITHUB_TOKEN
base=$(git merge-base "$base_ref" HEAD)
event=$(mktemp)
jq -n --arg b "$base" --arg h "$sha" --arg ref "$(git rev-parse --abbrev-ref HEAD)" --arg br "${base_ref#origin/}" \
  '{pull_request: {number: 0, base: {sha: $b, ref: $br}, head: {sha: $h, ref: $ref}, labels: []}}' > "$event"

start=$(date +%s); falhou=()
for f in "${fluxos[@]}"; do
  n=$(basename "$f"); log="$logs/$sha-$n.log"
  echo "▶ $n — log em ${log#"$root"/}"
  set +e
  "$act" pull_request -W "$f" -e "$event" \
    -P "ubuntu-latest=$imagem_local" -P "ubuntu-24.04=$imagem_local" -P "ubuntu-26.04=$imagem_local" \
    --container-architecture linux/amd64 --pull=false --var LOCAL_CI=true \
    -s GITHUB_TOKEN --privileged --rm --concurrent-jobs 2 \
    --container-options "--cpus=${CI_LOCAL_CPUS:-2} --memory=${CI_LOCAL_MEMORIA:-3g}" \
    --artifact-server-path "$root/.cache/ci-local/artifacts" 2>&1 | tee "$log" | grep -E --line-buffered '🏁|::error'
  rc=${PIPESTATUS[0]}
  set -e
  if [ "$rc" -ne 0 ] || grep -q 'Job failed' "$log"; then falhou+=("$n"); fi
done
dur=$(( $(date +%s) - start ))
if [ ${#falhou[@]} -gt 0 ]; then
  echo "❌ CI local FALHOU em ${dur}s: ${falhou[*]} — corrija antes do push (logs em .cache/ci-local/logs/)"; exit 1
fi
if [ -z "$so" ]; then
  { date -u +%FT%TZ; echo "duração=${dur}s act=$ACT_VERSION workflows=${#fluxos[@]}"; } > "$ok"
fi
echo "✅ CI local verde em ${dur}s para $(git rev-parse --short HEAD) (${#fluxos[@]} workflows)"
