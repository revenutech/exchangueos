#!/usr/bin/env bash
# Proves .ci-local/passos.conf: for one synthetic commit per kind of change, `rodada-escopo.sh lista <base>` must put in
# the N1 exactly the expected steps. Negative control: a change only under .base/plans/ must leave every Go step out
# (only gitleaks enters); if the keys were too wide, it would fail. Commits are built with plumbing (temporary index +
# commit-tree), so no hook runs; `lista` reads HEAD, so each case runs in a throwaway detached worktree.
# Usage: tests/ci-local/passos-controle-negativo.sh [<base-ref>]
#   driver from $RODADA_ESCOPO or ~/projects/revenu-platform/baremetal/scripts/ci-local-servico/rodada-escopo.sh
set -euo pipefail

driver=${RODADA_ESCOPO:-$HOME/projects/revenu-platform/baremetal/scripts/ci-local-servico/rodada-escopo.sh}
[ -f "$driver" ] || { echo "FAIL: driver missing: $driver"; exit 1; }
top=$(git rev-parse --show-toplevel)
cd "$top"
base=$(git rev-parse "${1:-HEAD}")
git cat-file -e "$base:.ci-local/passos.conf" || { echo "FAIL: .ci-local/passos.conf missing in $base"; exit 1; }

first_file() { # <prefix> <ext>: first tracked file whose path starts with <prefix> and ends in .<ext>
  git ls-tree -r --name-only "$base" | grep -m1 -E "^$1.*\.$2\$" || true
}

synthetic_commit() { # <path> <comment>: commit on top of base that appends <comment> to <path>
  local idx blob tree
  idx=$(mktemp)
  GIT_INDEX_FILE=$idx git read-tree "$base"
  blob=$({ git show "$base:$1"; printf '\n%s\n' "$2"; } | git hash-object -w --stdin)
  GIT_INDEX_FILE=$idx git update-index --cacheinfo "100644,$blob,$1"
  tree=$(GIT_INDEX_FILE=$idx git write-tree)
  rm -f "$idx"
  git -c user.name=passos-controle -c user.email=passos-controle@localhost commit-tree "$tree" -p "$base" -m "passos controle: $1"
}

fails=0
total=0
wt=""
cleanup() { if [ -n "$wt" ]; then git worktree remove --force "$wt" 2>/dev/null || rm -rf "$wt"; wt=""; fi; }
trap cleanup EXIT

# check <name> <path> <comment> <regex of steps that must enter> <regex of steps that must stay out> <min entering>
check() {
  local out head entram
  total=$((total + 1))
  if [ -z "$2" ]; then echo "FAIL $1 → no tracked file for the case"; fails=$((fails + 1)); return 0; fi
  head=$(synthetic_commit "$2" "$3")
  wt=$(mktemp -d)
  git worktree add -q --detach "$wt" "$head"
  # proto/gen is generated and ignored by git; `go list` needs it.
  if [ -d "$top/proto/gen" ]; then cp -R "$top/proto/gen" "$wt/proto/"; fi
  out=$(cd "$wt" && bash "$driver" lista "$base" 2>&1) || { echo "FAIL $1 → lista failed:"; printf '%s\n' "$out" | sed 's/^/     /'; fails=$((fails + 1)); cleanup; return 0; }
  cleanup
  entram=$(printf '%s\n' "$out" | awk '$1 == "entra" { print $2 }')
  local ok=1 n
  n=$(printf '%s\n' "$entram" | grep -c . || true)
  if [ -n "$4" ] && ! printf '%s\n' "$entram" | grep -Eq "$4"; then ok=0; echo "FAIL $1 → expected a step matching /$4/ to enter"; fi
  if [ -n "$5" ] && printf '%s\n' "$entram" | grep -Eq "$5"; then ok=0; echo "FAIL $1 → a step matching /$5/ entered: $(printf '%s\n' "$entram" | grep -E "$5" | paste -sd' ' -)"; fi
  if [ "$n" -lt "$6" ]; then ok=0; echo "FAIL $1 → $n steps entered, expected at least $6"; fi
  if [ "$ok" = 1 ]; then echo "ok   $1 → $n step(s) enter ($2)"; else fails=$((fails + 1)); fi
}

all=$(git show "$base:.ci-local/passos.conf" | grep -c ' -- ' || true)
# negative control: plan text only → gitleaks, nothing else
check "negativo .base/plans" "$(first_file .base/plans/milestones/backlog md)" "<!-- passos controle -->" '^gitleaks$' '^(teste-|build|govulncheck|bacen-mock|helm|escopo|passos|lint-patterns|shellcheck)' 1
# a leaf package (nobody imports pkg/sanctions): its own test step and the tree-wide Go steps; no disk gates
check "pkg/sanctions" "$(first_file pkg/sanctions/ go)" "// passos controle" '^teste-pkg_sanctions$' '^(helm|escopo|passos|lint-patterns|bacen-mock|teste-cmd_healthcheck)$' 4
# disk read declared in escopo.conf (le-disco pkg/bacen → data/bacen/): every Go test step enters
check "data/bacen" "$(first_file data/bacen/ '[a-z]+')" "# passos controle" '^teste-pkg_bacen$' '^(build|govulncheck|helm|escopo)$' 10
# chart only
check "helm" "$(first_file deploy/helm/exchangeos/ yaml)" "# passos controle" '^helm$' '^(teste-|build|govulncheck|escopo)' 2
# .ci-local/ widens: every line of passos.conf has at least one step entering
check ".ci-local amplia" ".ci-local/escopo.conf" "# passos controle" '^(escopo|passos|teste-pkg_sanctions|helm|gitleaks)$' '' "$all"

if [ "$fails" != 0 ]; then echo "passos: $fails falha(s)"; exit 1; fi
echo "passos: $total/$total"
