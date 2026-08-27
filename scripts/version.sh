#!/usr/bin/env bash
# 计算构建用的 VERSION / COMMIT / DATE / LDFLAGS。
# 发布 tag 精确匹配时用 tag；开发构建用 version.go 的默认值加 -dev。
if [ -n "${BASH_VERSION:-}" ]; then
  _ver_src="${BASH_SOURCE[0]}"
else
  _ver_src="$0"
fi
ROOT="$(cd "$(dirname "$_ver_src")/.." && pwd)"
PKG="github.com/ts721521/DevToolbox/internal/version"

# 干净工作区且 HEAD 正好是某个 tag → 用该 tag（正式发布构建）。
# 其余情况（有改动、或尚未打 tag）→ 用 version.go 并加 -dev。
if [ -z "$(git -C "$ROOT" status --porcelain)" ] && TAG=$(git -C "$ROOT" describe --tags --exact-match 2>/dev/null); then
  VERSION="$TAG"
else
  DEFAULT=$(sed -n 's/^[[:space:]]*Version = "\([^"]*\)".*/\1/p' "$ROOT/internal/version/version.go" | head -1)
  DEFAULT="${DEFAULT:-0.0.0-dev}"
  VERSION="v${DEFAULT#v}-dev"
fi
COMMIT=$(git -C "$ROOT" rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)
# Darwin: do not pass -s. It can strip LC_UUID and dyld aborts the binary (Abort trap 6).
_goos="${GOOS:-$(go env GOOS 2>/dev/null || true)}"
_strip="-s -w"
[ "$_goos" = darwin ] && _strip="-w"
LDFLAGS="${_strip} -X ${PKG}.Version=${VERSION} -X ${PKG}.Commit=${COMMIT} -X ${PKG}.Date=${DATE}"
export VERSION COMMIT DATE LDFLAGS PKG ROOT
