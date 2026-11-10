#!/usr/bin/env bash
# Proves .ci-local/escopo.conf: one synthetic commit per category must make `--nivel` answer «amplia» with that
# category in a «motivo:» line, and a commit outside every category must answer «N1» with no motivo (negative control).
# Commits are built with plumbing (temporary index + commit-tree), so no commit hook runs; --nivel needs the head
# checked out, so each case runs in a throwaway detached worktree. The caller's working tree is untouched.
# Usage: tests/ci-local/escopo-controle-negativo.sh [<base-ref>]   (gate from $REVENU_GATE or /private/tmp/revenu-gate-pesado.sh)
set -euo pipefail

gate=${REVENU_GATE:-/private/tmp/revenu-gate-pesado.sh}
cd "$(git rev-parse --show-toplevel)"
base=$(git rev-parse "${1:-HEAD}")
git cat-file -e "$base:.ci-local/escopo.conf" || { echo "FAIL: .ci-local/escopo.conf missing in $base"; exit 1; }

first_file() { # <prefix> <ext>: first tracked file whose path starts with <prefix> and ends in .<ext>
  git ls-tree -r --name-only "$base" | grep -m1 -E "^$1.*\.$2\$"
}

synthetic_commit() { # <path> <comment>: commit on top of base that appends <comment> to <path>
  local idx blob tree
  idx=$(mktemp)
  GIT_INDEX_FILE=$idx git read-tree "$base"
  blob=$({ git show "$base:$1"; printf '\n%s\n' "$2"; } | git hash-object -w --stdin)
  GIT_INDEX_FILE=$idx git update-index --cacheinfo "100644,$blob,$1"
  tree=$(GIT_INDEX_FILE=$idx git write-tree)
  rm -f "$idx"
  git -c user.name=escopo-controle -c user.email=escopo-controle@localhost commit-tree "$tree" -p "$base" -m "escopo controle: $1"
}

fails=0
total=0
wt=""
cleanup() { if [ -n "$wt" ]; then git worktree remove --force "$wt" 2>/dev/null || rm -rf "$wt"; wt=""; fi; }
trap cleanup EXIT

check() { # <expect: amplia|N1> <category or -> <path> <comment>
  local out head first
  total=$((total + 1))
  if [ -z "$3" ]; then
    echo "FAIL $2 → nenhum arquivo rastreado para o caso (first_file vazio)"; fails=$((fails + 1)); return 0
  fi
  head=$(synthetic_commit "$3" "$4")
  wt=$(mktemp -d)
  git worktree add -q --detach "$wt" "$head"
  out=$(cd "$wt" && "$gate" --nivel "$base" "$head") || out="--nivel falhou (rc $?)"$'\n'"$out"
  cleanup
  first=$(printf '%s\n' "$out" | sed -n 1p)
  if [ "$1" = amplia ]; then
    if [ "$first" = amplia ] && printf '%s\n' "$out" | grep -q "^motivo: $2: $3\$"; then
      echo "ok   $2 → amplia ($3)"
    else
      echo "FAIL $2 → esperado amplia com motivo $2 ($3); veio:"; printf '%s\n' "$out" | sed 's/^/     /'; fails=$((fails + 1))
    fi
  else
    if [ "$first" = N1 ] && ! printf '%s\n' "$out" | grep -q '^motivo:'; then
      echo "ok   negativo → N1 ($3)"
    else
      echo "FAIL negativo → esperado N1 sem motivo ($3); veio:"; printf '%s\n' "$out" | sed 's/^/     /'; fails=$((fails + 1))
    fi
  fi
}

check amplia dinheiro   "$(first_file modules/trade/domain go)"   "// escopo controle"
check amplia dinheiro   "$(first_file migrations/000002_create_fx_trades sql)" "-- escopo controle"
check amplia tenant     "$(first_file internal/adminapi go)"      "// escopo controle"
check amplia assinatura "$(first_file cmd/cred-rotator go)"       "// escopo controle"
check amplia pii        "$(first_file internal/erasure go)"       "// escopo controle"
check amplia contrato   "$(first_file proto/exchangeos/v1 proto)" "// escopo controle"
check amplia catalogo   "$(first_file .base/aasc ttl)"            "# escopo controle"
check N1     -          "$(first_file cmd/healthcheck go)"       "// escopo controle"

if [ "$fails" != 0 ]; then echo "escopo: $fails falha(s)"; exit 1; fi
echo "escopo: $total/$total"
