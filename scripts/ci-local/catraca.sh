#!/usr/bin/env bash
# Catraca de cobertura por pacote, head × base (CONVENTIONS § 6; «catraca contra base»): a N2 de um lote reprova se
# algum pacote com testes perde cobertura em relação à base do lote, ou some. Pacote novo entra sem piso.
# Roda na raiz do repositório, no head. A base é a do lote: CATRACA_BASE=<sha> (obrigatória; a N2 não tem base própria).
# A base é testada num worktree destacado e descartável; o proto/gen (ignorado pelo git) é copiado do head.
# Saída: uma linha por queda («QUEDA <pacote> <base> → <head>», «SUMIU <pacote> <base>»), os pacotes novos e o resumo.
# Sai 1 se houver queda, 2 em erro de uso.
set -uo pipefail
base=${CATRACA_BASE:-}
[ -n "$base" ] || { echo "catraca: CATRACA_BASE não definida (a base do lote)" >&2; exit 2; }
top=$(git rev-parse --show-toplevel) || exit 2
cd "$top" || exit 2
git rev-parse -q --verify "$base^{commit}" >/dev/null || { echo "catraca: base inválida: $base" >&2; exit 2; }
tmp=$(mktemp -d)
wt=$tmp/base
cleanup() { git worktree remove --force "$wt" >/dev/null 2>&1 || true; rm -rf "$tmp"; }
trap cleanup EXIT

cobertura() { # <dir> <saida.tsv>: «pacote cobertura» dos pacotes com testes, ordenado
  (cd "$1" && go test -count=1 -cover ./... 2>&1) | tee "$2.log" |
    awk '$1 == "ok" { for (i = 1; i <= NF; i++) if ($i == "coverage:") { sub(/%/, "", $(i + 1)); print $2, $(i + 1) } }' |
    sort > "$2"
}

git worktree add -q --detach "$wt" "$base" || exit 2
if [ -d proto/gen ]; then mkdir -p "$wt/proto" && cp -R proto/gen "$wt/proto/"; fi
cobertura "$wt" "$tmp/base.tsv"
cobertura "$top" "$tmp/head.tsv"
if grep -q '^FAIL' "$tmp/head.tsv.log"; then echo "catraca: há teste falhando no head"; grep '^FAIL' "$tmp/head.tsv.log" | head; exit 1; fi

quedas=$(join -a1 "$tmp/base.tsv" "$tmp/head.tsv" |
  awk 'NF == 3 && $3 + 0 < $2 + 0 { print "QUEDA", $1, $2, "→", $3 } NF == 2 { print "SUMIU", $1, $2 }')
join -v2 "$tmp/base.tsv" "$tmp/head.tsv" | sed 's/^/novo: /'
nb=$(wc -l < "$tmp/base.tsv" | tr -d ' '); nh=$(wc -l < "$tmp/head.tsv" | tr -d ' ')
if [ -n "$quedas" ]; then
  printf '%s\n' "$quedas"
  echo "catraca: $(printf '%s\n' "$quedas" | wc -l | tr -d ' ') queda(s) contra $base ($nb pacotes na base, $nh no head)"
  exit 1
fi
echo "catraca ok contra $base: $nb pacotes na base, $nh no head, nenhuma queda"
