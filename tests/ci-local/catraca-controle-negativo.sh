#!/usr/bin/env bash
# Proves scripts/ci-local/catraca.sh on a synthetic Go module (two packages, git history in a temp dir):
#   coverage kept or a new package          → passes (positive control);
#   uncovered code added to a package       → QUEDA, exit 1 (negative control);
#   a tested package removed                → SUMIU, exit 1;
#   no CATRACA_BASE                         → exit 2.
set -uo pipefail
catraca=$(cd "$(dirname "$0")/../.." && pwd)/scripts/ci-local/catraca.sh
tmp=$(mktemp -d); trap 'rm -rf "$tmp"' EXIT
export GOFLAGS='' GOWORK=off GOTOOLCHAIN=local
fails=0
confere() { # <esperado> <nome> <visto>
  if [ "$3" = "$1" ]; then echo "ok   $2"; else echo "FAIL $2: '$3', esperado '$1'"; fails=$((fails + 1)); fi
}
cd "$tmp" || exit 1
git init -q; git config user.email t@t; git config user.name t; git config commit.gpgsign false
mkdir a b
printf 'module ex.com/m\n\ngo 1.22\n' > go.mod
printf 'package a\n\nfunc A() int { return 1 }\n' > a/a.go
printf 'package a\n\nimport "testing"\n\nfunc TestA(t *testing.T) { if A() != 1 { t.Fatal() } }\n' > a/a_test.go
printf 'package b\n\nfunc B() int { return 2 }\n' > b/b.go
printf 'package b\n\nimport "testing"\n\nfunc TestB(t *testing.T) { if B() != 2 { t.Fatal() } }\n' > b/b_test.go
git add -A; git commit -qm base; base=$(git rev-parse HEAD)

roda() { CATRACA_BASE=$1 bash "$catraca" > "$tmp/out" 2>&1; echo $?; }

confere 0 'mesma cobertura passa' "$(roda "$base")"

mkdir c
printf 'package c\n\nfunc C() int { return 3 }\n' > c/c.go
git add -A; git commit -qm novo
confere 0 'pacote novo sem teste não reprova' "$(roda "$base")"

printf 'package a\n\nfunc A() int { return 1 }\n\nfunc Unused(x int) int {\n\tif x > 0 {\n\t\treturn x\n\t}\n\treturn -x\n}\n' > a/a.go
git commit -qam queda
confere 1 'código sem teste no pacote a reprova' "$(roda "$base")"
confere 1 'a saída diz QUEDA do pacote a' "$(grep -c '^QUEDA ex.com/m/a ' "$tmp/out")"

git checkout -q "$base" -- a; git rm -rq b; git commit -qm sumiu
confere 1 'pacote testado que some reprova' "$(roda "$base")"
confere 1 'a saída diz SUMIU do pacote b' "$(grep -c '^SUMIU ex.com/m/b ' "$tmp/out")"

confere 2 'sem CATRACA_BASE sai 2' "$(roda '')"

if [ "$fails" != 0 ]; then echo "catraca: $fails falha(s)"; exit 1; fi
echo "catraca: controle 7/7"
