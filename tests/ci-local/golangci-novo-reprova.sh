#!/usr/bin/env bash
# Negative control of the golangci-lint ratchet (issues.new-from-rev in .golangci.yml; RPTN.091.026.01,
# MS-024i 1.32). In a throwaway clone of HEAD:
#   1. positive control: golangci-lint on ./pkg/outbox/... passes (old debt before new-from-rev is not reported);
#   2. a new commit adds a file with an unchecked error; golangci-lint must FAIL and name that file.
# If step 2 passes, the ratchet is blind (new-from-rev too new, or the linter set dropped errcheck) and this
# script exits 1.
# Needs golangci-lint v2 on PATH (the version pinned in .github/workflows/ci.yml).
set -euo pipefail

root=$(git rev-parse --show-toplevel)
lint=${GOLANGCI_LINT:-golangci-lint}
command -v "$lint" >/dev/null || { echo "golangci-novo-reprova: $lint not found (install the version pinned in ci.yml)" >&2; exit 2; }

# Physical path: on macOS mktemp lives under /var -> /private/var, and new-from-rev then fails to match the diff
# paths and drops every issue (a blind ratchet that this control would wrongly blame on the config).
tmp=$(cd "$(mktemp -d)" && pwd -P)
trap 'rm -rf "$tmp"' EXIT
git clone -q --no-hardlinks "$root" "$tmp/repo"
cd "$tmp/repo"
git checkout -q "$(git -C "$root" rev-parse HEAD)"

if ! "$lint" run --timeout 5m ./pkg/outbox/... >"$tmp/pos.log" 2>&1; then
  echo "FAIL positive control: golangci-lint reported issues on unchanged HEAD (see below)" >&2
  cat "$tmp/pos.log" >&2
  exit 1
fi

cat > pkg/outbox/zz_ratchet_negativo.go <<'GO'
package outbox

import "os"

// ratchetNegativo exists only in the negative control: errcheck must flag the dropped error.
func ratchetNegativo() {
	os.Remove("/nonexistent")
}
GO
git -c user.name=ci -c user.email=ci@localhost add pkg/outbox/zz_ratchet_negativo.go
git -c user.name=ci -c user.email=ci@localhost commit -q -m "test: new lint issue"

if "$lint" run --timeout 5m ./pkg/outbox/... >"$tmp/neg.log" 2>&1; then
  echo "FAIL negative control: a new unchecked error passed the ratchet" >&2
  cat "$tmp/neg.log" >&2
  exit 1
fi
grep -q 'zz_ratchet_negativo.go' "$tmp/neg.log" || {
  echo "FAIL negative control: golangci-lint failed, but not on the new file" >&2
  cat "$tmp/neg.log" >&2
  exit 1
}
echo "ok: old debt ignored, new issue refused"
