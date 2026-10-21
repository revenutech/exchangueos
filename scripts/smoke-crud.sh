#!/usr/bin/env bash
# scripts/smoke-crud.sh — verifies every admin-API table returns ≥ MIN_ROWS rows.
#
# Drives the /v1/admin/_schemas discovery endpoint to enumerate exposed tables,
# then LISTs each one and asserts count >= MIN_ROWS (default 5). Single failing
# table aborts with non-zero exit so CI can catch seed regressions.
#
# Env:
#   EXCHANGEOS_BASE_URL  default http://localhost:8084
#   MIN_ROWS             default 5

set -euo pipefail

BASE_URL="${EXCHANGEOS_BASE_URL:-http://localhost:8084}"
MIN_ROWS="${MIN_ROWS:-5}"

echo "── ExchangeOS CRUD smoke (≥ ${MIN_ROWS} rows per table) ──"
echo "  base URL : ${BASE_URL}"
echo ""

# Fetch the schema catalogue.
catalogue=$(curl -fsS --max-time 5 "${BASE_URL}/v1/admin/_schemas") || {
    echo "❌ failed to fetch /_schemas — is the api up + EXCHANGEOS_ENABLE_ADMIN_API=true?" >&2
    exit 1
}

# Extract URLs via portable grep/sed (no jq dependency).
urls=$(echo "${catalogue}" | grep -oE '"url":"[a-z_-]+"' | sed 's/"url":"//; s/"//')

total=0
fails=0

while IFS= read -r url; do
    [[ -z "$url" ]] && continue
    total=$((total + 1))
    # Use unlimited limit so we can verify total table size (LIST is page-bounded by default 100).
    body=$(curl -fsS --max-time 5 "${BASE_URL}/v1/admin/${url}?limit=500" || echo '{"count":-1}')
    count=$(echo "${body}" | grep -oE '"count":[0-9-]+' | head -1 | sed 's/"count"://')
    if [[ "${count}" -ge "${MIN_ROWS}" ]]; then
        printf "  ✅  %-22s %d rows\n" "${url}" "${count}"
    else
        printf "  ❌  %-22s %d rows (want ≥ %d)\n" "${url}" "${count}" "${MIN_ROWS}"
        fails=$((fails + 1))
    fi
done <<< "${urls}"

echo ""
if [[ ${fails} -eq 0 ]]; then
    echo "✅ All ${total} tables satisfy minimum ${MIN_ROWS} rows."
    exit 0
fi
echo "❌ ${fails} of ${total} tables below ${MIN_ROWS} rows. Re-run seeds or extend them." >&2
exit 1
