#!/usr/bin/env bash
# Instala o `act` pinado (tools.lock) em .cache/ci-local/bin, verificando sha256. Idempotente.
set -euo pipefail
root=$(git rev-parse --show-toplevel); here="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=/dev/null
. "$here/tools.lock"
bin="$root/.cache/ci-local/bin"; mkdir -p "$bin"
if [ -x "$bin/act" ] && "$bin/act" --version 2>/dev/null | grep -q "$ACT_VERSION"; then echo "$bin/act"; exit 0; fi
os=$(uname -s); arch=$(uname -m); [ "$arch" = aarch64 ] && arch=arm64; [ "$arch" = amd64 ] && arch=x86_64
var="ACT_SHA256_${os}_${arch}"; want="${!var:-}"
[ -n "$want" ] || { echo "act: plataforma $os/$arch sem sha256 em tools.lock" >&2; exit 1; }
tmp=$(mktemp -d); trap 'rm -rf "$tmp"' EXIT
curl -sSfL --retry 3 -o "$tmp/act.tgz" "https://github.com/nektos/act/releases/download/v${ACT_VERSION}/act_${os}_${arch}.tar.gz"
got=$(shasum -a 256 "$tmp/act.tgz" | cut -d' ' -f1)
[ "$got" = "$want" ] || { echo "act: sha256 inválido ($got ≠ $want)" >&2; exit 1; }
tar -xzf "$tmp/act.tgz" -C "$tmp" act && install -m 0755 "$tmp/act" "$bin/act"
echo "$bin/act"
