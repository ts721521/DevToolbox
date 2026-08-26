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
ASSET="devtoolbox-${goos}-${goarch}-${TAG}${ext}"
DEST="${HOME}/bin/devtoolbox${ext}"
mkdir -p "$(dirname "$DEST")"

if [ "$FORCE" != true ] && [ -x "$DEST" ]; then
  if "$DEST" version 2>/dev/null | grep -q "$TAG"; then
    echo "已是 $TAG"
    exit 0
  fi
fi

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
gh release download "$TAG" --repo "$REPO" --pattern "$ASSET" --dir "$TMP"
install -m 755 "$TMP/$ASSET" "$DEST"
echo "已安装 $DEST ($TAG)"
"$DEST" install-desktop || true
echo "运行: devtoolbox   或双击桌面「开发工具箱」"
