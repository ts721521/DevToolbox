#!/usr/bin/env bash
# 本地构建带版本号的二进制。输出到 dist/tooldock-<os>-<arch>-<tag>
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
# shellcheck source=version.sh
source "$ROOT/scripts/version.sh"

cd "$ROOT"
GOOS="${GOOS:-$(go env GOOS)}"
GOARCH="${GOARCH:-$(go env GOARCH)}"
ext=""
[ "$GOOS" = windows ] && ext=".exe"
mkdir -p dist
OUT="dist/tooldock-${GOOS}-${GOARCH}-${VERSION}${ext}"
CGO_ENABLED="${CGO_ENABLED:-0}" go build -trimpath -ldflags "$LDFLAGS" -o "$OUT" .
echo "$OUT"
