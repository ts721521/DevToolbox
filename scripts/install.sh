#!/usr/bin/env bash
# 从 GitHub Releases 安装最新版到 ~/bin，并可选装到桌面。
set -euo pipefail
REPO="${DEVTOOLBOX_REPO:-ts721521/DevToolbox}"
FORCE=false
for arg in "$@"; do
  case "$arg" in
    -f|--force) FORCE=true ;;
  esac
done

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"
case "$os" in
  darwin) goos=darwin ;;
  linux) goos=linux ;;
  mingw*|msys*|cygwin*) goos=windows ;;
  *) echo "不支持的系统: $os"; exit 1 ;;
esac
case "$arch" in
  arm64|aarch64) goarch=arm64 ;;
  x86_64|amd64) goarch=amd64 ;;
  *) echo "不支持的架构: $arch"; exit 1 ;;
esac

ext=""
[ "$goos" = windows ] && ext=".exe"

TAG=$(gh release view --repo "$REPO" --json tagName --jq .tagName)
DEST="${HOME}/bin/tooldock${ext}"
mkdir -p "$(dirname "$DEST")"

if [ "$FORCE" != true ] && [ -x "$DEST" ]; then
  if "$DEST" version 2>/dev/null | grep -q "$TAG"; then
    echo "已是 $TAG"
    exit 0
  fi
fi

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
ASSET=""
for name in \
  "tooldock-${goos}-${goarch}-${TAG}${ext}" \
  "devtoolbox-${goos}-${goarch}-${TAG}${ext}"; do
  if gh release download "$TAG" --repo "$REPO" --pattern "$name" --dir "$TMP" 2>/dev/null; then
    ASSET="$name"
    break
  fi
done
if [ -z "$ASSET" ]; then
  echo "未找到 $TAG 的 ${goos}/${goarch} 安装包"
  exit 1
fi

SUMS="SHA256SUMS-${TAG}.txt"
if gh release download "$TAG" --repo "$REPO" --pattern "$SUMS" --dir "$TMP" 2>/dev/null; then
  line=$(grep -F " ${ASSET}" "$TMP/$SUMS" || true)
  if [ -z "$line" ]; then
    echo "✗ $SUMS 里没有 $ASSET"
    exit 1
  fi
  printf '%s\n' "$line" > "$TMP/check.txt"
  if command -v sha256sum >/dev/null 2>&1; then
    (cd "$TMP" && sha256sum -c check.txt)
  else
    (cd "$TMP" && shasum -a 256 -c check.txt)
  fi
else
  echo "警告: Release 没有 $SUMS，跳过校验（旧版本）"
fi

install -m 755 "$TMP/$ASSET" "$DEST"
echo "已安装 $DEST ($TAG)"
"$DEST" install-desktop || true
echo "运行: tooldock   或双击「工坞」"
