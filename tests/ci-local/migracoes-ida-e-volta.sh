#!/usr/bin/env bash
# Round trip of migrations/ on a throwaway single-node CockroachDB with TLS (verify-full; no --insecure), through
# cmd/migrator (golang-migrate, the production path):
#   1. up (all) and the checks of every tests/ci-local/migracoes/<NNNNNN>.check.sql must pass;
#   2. idempotence, for each migration that has a check file (new migrations bring one; the older ones predate the
#      rule and some use ADD CONSTRAINT without IF NOT EXISTS): its up.sql applied a second time, and its down.sql
#      twice in a row, succeed;
#   3. down (all) and then the checks must FAIL (negative control: a check that passes on an empty schema is vacuous);
#   4. up (all) again and the checks pass again.
# A check file holds one query per line (# comments) that must return exactly «t».
# Runs in the queue's DB lane (no REVENU_GATE_SEM_BANCO): the container carries label revenu.gate.dono.
# Env: CRDB_IMAGE (cockroachdb/cockroach:latest-v24.3), REVENU_GATE_DONO (ExchangueOS).
set -euo pipefail

top=$(git rev-parse --show-toplevel)
cd "$top"
image=${CRDB_IMAGE:-cockroachdb/cockroach:latest-v24.3}
dono=${REVENU_GATE_DONO:-ExchangueOS}
tmp=$(mktemp -d)
cid=""
cleanup() { if [ -n "$cid" ]; then docker rm -f "$cid" >/dev/null 2>&1 || true; fi; rm -rf "$tmp"; }
trap cleanup EXIT

cid=$(docker run -d --label "revenu.gate.dono=$dono" -p 127.0.0.1::26257 --entrypoint /bin/bash "$image" -c '
  set -e
  c=/cockroach/cockroach; d=/certs; mkdir -p $d
  $c cert create-ca --certs-dir=$d --ca-key=$d/ca.key
  $c cert create-node localhost 127.0.0.1 --certs-dir=$d --ca-key=$d/ca.key
  $c cert create-client root --certs-dir=$d --ca-key=$d/ca.key
  exec $c start-single-node --certs-dir=$d --listen-addr=:26257 --http-addr=127.0.0.1:8080')
sqlc() { docker exec -i "$cid" /cockroach/cockroach sql --certs-dir=/certs --format=tsv "$@"; }
for _ in $(seq 1 60); do sqlc -e 'SELECT 1' >/dev/null 2>&1 && break; sleep 1; done
sqlc -e 'SELECT 1' >/dev/null || { echo "FAIL: CockroachDB did not come up"; docker logs "$cid" | tail -20; exit 1; }
sqlc -e 'CREATE DATABASE IF NOT EXISTS exchangeos' >/dev/null
for f in ca.crt client.root.crt client.root.key; do docker cp -q "$cid:/certs/$f" "$tmp/$f"; done
chmod 600 "$tmp/client.root.key"
port=$(docker port "$cid" 26257/tcp | head -1 | sed 's/.*://')
export EXCHANGEOS_DB_DSN="postgres://root@localhost:$port/exchangeos?sslmode=verify-full&sslrootcert=$tmp/ca.crt&sslcert=$tmp/client.root.crt&sslkey=$tmp/client.root.key"
go build -o "$tmp/migrator" ./cmd/migrator
migrator() { "$tmp/migrator" "$@" > "$tmp/migrator.log" 2>&1 || { echo "FAIL: migrator $*"; tail -20 "$tmp/migrator.log"; exit 1; }; }

ups=$(find migrations -maxdepth 1 -name '*.up.sql' | sort)
n=$(printf '%s\n' "$ups" | wc -l | tr -d ' ')
idem=()
for c in tests/ci-local/migracoes/*.check.sql; do
  [ -e "$c" ] || continue
  u=$(find migrations -maxdepth 1 -name "$(basename "$c" .check.sql)_*.up.sql")
  [ -n "$u" ] || { echo "FAIL: $c has no migration"; exit 1; }
  idem+=("$u")
done
fails=0

checks() { # <expect: pass|fail>: run every check file; count the files that do not behave as expected
  local f q out bad=0 file_ok name
  for f in tests/ci-local/migracoes/*.check.sql; do
    [ -e "$f" ] || continue
    file_ok=1; name=${f##*/}
    while IFS= read -r q || [ -n "$q" ]; do
      case "$q" in ''|'#'*) continue ;; esac
      out=$(sqlc -d exchangeos -e "$q" 2>/dev/null </dev/null | sed -n 2p) || out=""
      [ "$out" = t ] || file_ok=0
      if [ "$1" = pass ] && [ "$out" != t ]; then echo "FAIL check $name: «$q» returned «$out»"; fi
    done < "$f"
    if [ "$1" = pass ] && [ "$file_ok" = 0 ]; then bad=$((bad + 1)); fi
    if [ "$1" = fail ] && [ "$file_ok" = 1 ]; then echo "FAIL check $name passes on an empty schema (vacuous)"; bad=$((bad + 1)); fi
  done
  return "$bad"
}

migrator up
checks pass || fails=$((fails + $?))
echo "ok   up: $n migrations"

for f in "${idem[@]}"; do sqlc -d exchangeos < "$f" >/dev/null 2>"$tmp/err" || { echo "FAIL idempotence: $f applied twice"; cat "$tmp/err"; fails=$((fails + 1)); }; done
checks pass || fails=$((fails + $?))
echo "ok   idempotence: up.sql applied twice: ${idem[*]}"

migrator down "$n"
checks fail || fails=$((fails + $?))
echo "ok   down $n: checks fail on the empty schema (negative control)"
for ((i = ${#idem[@]} - 1; i >= 0; i--)); do f=${idem[$i]}; d=${f%.up.sql}.down.sql; sqlc -d exchangeos < "$d" >/dev/null 2>"$tmp/err" || { echo "FAIL idempotence: $d on an already rolled back schema"; cat "$tmp/err"; fails=$((fails + 1)); }; done
echo "ok   idempotence: down.sql applied on an already rolled back schema"

migrator up
checks pass || fails=$((fails + $?))
echo "ok   up again: $n migrations"

if [ "$fails" != 0 ]; then echo "migracoes: $fails falha(s)"; exit 1; fi
echo "migracoes: ida e volta verde ($n migrações, $(cat tests/ci-local/migracoes/*.check.sql 2>/dev/null | grep -vc -E '^(#|$)') checks)"
