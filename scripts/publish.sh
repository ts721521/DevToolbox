#!/usr/bin/env bash
# 在 main 上打 tag 并推送，触发 GitHub Actions 发布。
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"
REPO="ts721521/DevToolbox"

VERSION=""
ASSUME_YES=false
for arg in "$@"; do
  case "$arg" in
    -y|--yes) ASSUME_YES=true ;;
    v*) VERSION="$arg" ;;
  esac
done

echo "════════════════════════════════════════"
echo "  DevToolbox 发布"
echo "════════════════════════════════════════"

BRANCH=$(git rev-parse --abbrev-ref HEAD)
if [ "$BRANCH" != "main" ]; then
  echo "✗ 必须在 main 分支发布(当前: $BRANCH)"
  exit 1
fi
if [ -n "$(git status --porcelain)" ]; then
  echo "✗ 工作区有未提交改动"
  git status --short
  exit 1
fi
git fetch origin
if [ "$(git rev-parse HEAD)" != "$(git rev-parse origin/main)" ]; then
  echo "✗ 本地 main 不是最新，先 git pull"
  exit 1
fi

if [ -z "$VERSION" ]; then
  LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
  BASE="${LATEST_TAG#v}"
  IFS='.' read -r MAJOR MINOR PATCH <<< "$BASE"
  PATCH=$((PATCH + 1))
  VERSION="v${MAJOR}.${MINOR}.${PATCH}"
  echo "→ 最新 tag: $LATEST_TAG → 推断: $VERSION"
fi
if ! echo "$VERSION" | grep -qE '^v[0-9]+\.[0-9]+\.[0-9]+$'; then
  echo "✗ 版本号应为 vX.Y.Z"
  exit 1
fi
if git rev-parse "$VERSION" >/dev/null 2>&1; then
  echo "✗ tag $VERSION 已存在"
  exit 1
fi

echo "即将发布 $VERSION"
if [ "$ASSUME_YES" = false ]; then
  read -r -p "确认发布? (y/N) " confirm
  [ "$confirm" = "y" ] || exit 0
fi

echo "→ 本地预检..."
go test ./...
go vet ./...
echo "✓ 预检通过"

git tag -a "$VERSION" -m "$VERSION"
git push origin "$VERSION"
echo "✓ 已推送 $VERSION"
echo "  查看: https://github.com/${REPO}/actions"
echo "  Release: https://github.com/${REPO}/releases/tag/${VERSION}"
