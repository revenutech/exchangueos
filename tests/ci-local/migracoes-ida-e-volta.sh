#!/usr/bin/env bash
# Round trip of migrations/ on a throwaway single-node CockroachDB with TLS (verify-full; no --insecure), through
# cmd/migrator (golang-migrate, the production path):
#   1. up (all) and the checks of every tests/ci-local/migracoes/<NNNNNN>.check.sql must pass;
#   2. idempotence, for each migration that has a check file (new migrations bring one; the older ones predate the
#      rule and some use ADD CONSTRAINT without IF NOT EXISTS): its up.sql applied a second time, and its down.sql
#      applied again right after its own rollback (one step down at a time), succeed;
#   3. down (all) and then the checks must FAIL (negative control: a check that passes on an empty schema is vacuous);
#   4. up (all) again and the checks pass again.
# A check file holds one query per line (# comments) that must return exactly «t».
# A tests/ci-local/migracoes/<NNNNNN>.recusa.sql holds «<constraint> | <sql>» per line: the sql must FAIL and the
# error must name <constraint>; «ok | <sql>» must succeed (positive control: the fixture itself is valid, so a
# refusal is the constraint's, not a broken fixture's). Each sql runs alone; end it with ROLLBACK to leave no row.
# After the final up, every package listed in tests/ci-local/migracoes/go-integracao.txt runs its integration tests
# (go test -tags integration) against this database; it must PASS at least one test and SKIP none (a skip there means
# the DSN did not reach the test, which would pass vacuously).
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

recusas() { # every line of every recusa file behaves as declared; prints a summary
  local f ln want sql out rc bad=0 total=0
  for f in tests/ci-local/migracoes/*.recusa.sql; do
    [ -e "$f" ] || continue
    while IFS= read -r ln || [ -n "$ln" ]; do
      case "$ln" in ''|'#'*) continue ;; esac
      want=${ln%% | *}; sql=${ln#* | }; total=$((total + 1))
      rc=0; out=$(sqlc -d exchangeos -e "$sql" 2>&1 </dev/null) || rc=$?
      if [ "$want" = ok ]; then
        [ "$rc" = 0 ] || { echo "FAIL ${f##*/}: expected success, got: $(printf '%s' "$out" | grep -m1 -i error)"; bad=$((bad + 1)); }
      elif [ "$rc" = 0 ]; then
        echo "FAIL ${f##*/}: $want did not refuse: ${sql:0:120}"; bad=$((bad + 1))
      elif ! printf '%s' "$out" | grep -q "$want"; then
        echo "FAIL ${f##*/}: refused, but not by $want: $(printf '%s' "$out" | grep -m1 -i error)"; bad=$((bad + 1))
      fi
    done < "$f"
  done
  echo "ok   recusas: $((total - bad))/$total"
  return "$bad"
}

migrator up
checks pass || fails=$((fails + $?))
recusas || fails=$((fails + $?))
echo "ok   up: $n migrations"

for f in "${idem[@]}"; do sqlc -d exchangeos < "$f" >/dev/null 2>"$tmp/err" || { echo "FAIL idempotence: $f applied twice"; cat "$tmp/err"; fails=$((fails + 1)); }; done
checks pass || fails=$((fails + $?))
echo "ok   idempotence: up.sql applied twice: ${idem[*]}"

# down one step at a time; right after a migration with a check file is rolled back, its down.sql runs once more on
# the very schema it belongs to (a down of an ALTER cannot run once the table under it is gone)
all=()
while IFS= read -r u; do all+=("$u"); done <<< "$ups"
for ((i = ${#all[@]} - 1; i >= 0; i--)); do
  migrator down 1
  for u in "${idem[@]}"; do
    [ "$u" = "${all[$i]}" ] || continue
    d=${u%.up.sql}.down.sql
    sqlc -d exchangeos < "$d" >/dev/null 2>"$tmp/err" || { echo "FAIL idempotence: $d applied twice"; cat "$tmp/err"; fails=$((fails + 1)); }
  done
done
echo "ok   idempotence: down.sql applied twice, each right after its own rollback"
checks fail || fails=$((fails + $?))
echo "ok   down $n: checks fail on the empty schema (negative control)"

migrator up
checks pass || fails=$((fails + $?))
echo "ok   up again: $n migrations"

if [ -f tests/ci-local/migracoes/go-integracao.txt ]; then
  while IFS= read -r line || [ -n "$line" ]; do
    case "$line" in ''|'#'*) continue ;; esac
    # «<pkg>» or «<pkg> -run <regex>»: the filter is for packages that mix DB tests with tests of the live server.
    read -r pkg run <<< "$line"
    run_args=()
    if [ -n "$run" ]; then
      case "$run" in "-run "*) run_args=(-run "${run#-run }") ;; *) echo "FAIL go-integracao.txt: «$line»"; fails=$((fails + 1)); continue ;; esac
    fi
    if EXCHANGEOS_TEST_DSN="$EXCHANGEOS_DB_DSN" go test -tags integration -count=1 -v ${run_args[@]+"${run_args[@]}"} "$pkg" > "$tmp/go.log" 2>&1 </dev/null \
      && grep -q -- '--- PASS' "$tmp/go.log" && ! grep -q -- '--- SKIP' "$tmp/go.log"; then
      # The go test -v result lines go to stdout: the driver counts them on a banco step (MS-023 1.39, item 13).
      grep -E -- '^(=== RUN|--- (PASS|FAIL|SKIP)|ok |FAIL|PASS$)' "$tmp/go.log"
      echo "ok   go integration: $pkg ($(grep -c -- '--- PASS' "$tmp/go.log") passed)"
    else
      echo "FAIL go integration: $pkg"; grep -E -- '--- (FAIL|SKIP)|Error|error' "$tmp/go.log" | head -30; tail -5 "$tmp/go.log"
      fails=$((fails + 1))
    fi
  done < tests/ci-local/migracoes/go-integracao.txt
fi

if [ "$fails" != 0 ]; then echo "migracoes: $fails falha(s)"; exit 1; fi
echo "migracoes: ida e volta verde ($n migrações, $(cat tests/ci-local/migracoes/*.check.sql 2>/dev/null | grep -vc -E '^(#|$)') checks)"
