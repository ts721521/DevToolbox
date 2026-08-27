# 版本发布

## SemVer

`MAJOR.MINOR.PATCH`（tag 形如 `v1.2.0`）

| 位 | 何时 |
|----|------|
| MAJOR | 破坏性 CLI / JSON schema / API 变更 |
| MINOR | 向下兼容新功能 |
| PATCH | 向下兼容修复 |

源码默认版本在 `internal/version/version.go`。推送 tag 后，CI 用 ldflags 把 **tag 全名**注入二进制，界面 / `tooldock version` / 安装包文件名都用这个号。

## 发版前

- [ ] `go test ./...`、`go vet ./...` 通过
- [ ] CHANGELOG `[未发布]` 已挪到新版本段落（如 `## [1.2.0]`）
- [ ] `internal/version.Version` 与 tag 一致（`1.2.0` 或 `v1.2.0`）
- [ ] `./scripts/publish.sh` 会再检查上述两项

## 推荐流程

```bash
git checkout main && git pull
./scripts/publish.sh v1.2.0
```

脚本会：工作区检查 → CHANGELOG / version.go 核对 → 本地测试 → 打 annotated tag → 推送 → GitHub Actions 交叉编译并创建 Release。

手动等价：

```bash
git tag -a v1.2.0 -m "v1.2.0"
git push origin v1.2.0
```

推送 `v*` tag 会触发 `.github/workflows/release.yml`。Release 标题为 **工坞 ToolDock vX.Y.Z**，产物：

| 文件 | 说明 |
|------|------|
| `tooldock-darwin-arm64-vX.Y.Z` | macOS Apple Silicon 二进制 |
| `tooldock-darwin-amd64-vX.Y.Z` | macOS Intel 二进制 |
| `tooldock-windows-amd64-vX.Y.Z.exe` | Windows x64 |
| `tooldock-windows-arm64-vX.Y.Z.exe` | Windows ARM |
| `tooldock-linux-amd64-vX.Y.Z` | Linux x64（辅助） |
| `ToolDock-vX.Y.Z-windows-<arch>.exe` | Windows 带产品名的副本 |
| `ToolDock-vX.Y.Z-macOS-arm64.zip` | 内含 `工坞-vX.Y.Z.app` |
| `devtoolbox-*-vX.Y.Z` | 兼容旧文件名 |
| `SHA256SUMS-vX.Y.Z.txt` | 校验和 |

`scripts/install.sh` 会下载对应平台二进制并用 `SHA256SUMS-vX.Y.Z.txt` 校验；旧 Release 没有该文件时只警告、仍安装。

本地封装（会把当前构建的版本写进文件名）：

```bash
make pack
# 或
./scripts/build.sh && ./dist/tooldock-$(go env GOOS)-$(go env GOARCH)-v1.2.0-dev pack --out dist
```

## 用户安装

见 [Releases](https://github.com/ts721521/DevToolbox/releases) 或 `scripts/install.sh`。
