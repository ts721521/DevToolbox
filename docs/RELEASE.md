# 版本发布

## SemVer

`MAJOR.MINOR.PATCH`（tag 形如 `v1.0.0`）

| 位 | 何时 |
|----|------|
| MAJOR | 破坏性 CLI / JSON schema / API 变更 |
| MINOR | 向下兼容新功能 |
| PATCH | 向下兼容修复 |

## 发版前

- [ ] `go test ./...`、`go vet ./...` 通过
- [ ] CHANGELOG `[未发布]` 已挪到新版本
- [ ] `internal/version.Version` 与 tag 一致（CI 还会用 ldflags 注入）

## 推荐流程

```bash
git checkout main && git pull
./scripts/publish.sh v1.1.0
```

脚本会：工作区检查 → 本地测试 → 打 annotated tag → 推送 → GitHub Actions 交叉编译并创建 Release。

手动等价：

```bash
git tag -a v1.1.0 -m "v1.1.0"
git push origin v1.1.0
```

推送 `v*` tag 会触发 `.github/workflows/release.yml`，产物：

- `devtoolbox-darwin-arm64-vX.Y.Z`
- `devtoolbox-darwin-amd64-vX.Y.Z`
- `devtoolbox-windows-amd64-vX.Y.Z.exe`
- `devtoolbox-windows-arm64-vX.Y.Z.exe`
- `devtoolbox-linux-amd64-vX.Y.Z`

## 用户安装

见 [Releases](https://github.com/ts721521/DevToolbox/releases) 或 `scripts/install.sh`。
