#!/usr/bin/env bash
# Positive and negative control for scripts/lint-patterns.sh (MS-024k 1.2).
set -u
cd "$(dirname "$0")/../.." || exit 2
lint=scripts/lint-patterns.sh
dir=tests/scripts/testdata/lint-patterns
fail=0

if out=$("$lint" "$dir/completo.md" 2>&1); then
  echo "ok: complete pattern passes"
else
  echo "FAIL: complete pattern rejected:"; echo "$out"; fail=1
fi

if out=$("$lint" "$dir/incompleto.md" 2>&1); then
  echo "FAIL: incomplete pattern accepted"; fail=1
else
  for want in "Anti-pattern section missing" "no fenced code example" "Related has no cross-link"; do
    case "$out" in
      *"$want"*) echo "ok: incomplete pattern rejected ($want)" ;;
      *) echo "FAIL: missing diagnostic: $want"; echo "$out"; fail=1 ;;
    esac
  done
fi

if "$lint" >/dev/null 2>&1; then echo "FAIL: no args accepted"; fail=1; else echo "ok: no args rejected"; fi
exit "$fail"
