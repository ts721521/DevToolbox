# 贡献指南

感谢参与 **开发工具箱 (DevToolbox)**。提交代码前请读完本页和 `docs/` 下的规范。

## 快速开始

```bash
git clone https://github.com/ts721521/DevToolbox.git
cd DevToolbox
go test ./...
go build -o devtoolbox .
./devtoolbox
```

提交 PR 前：

```bash
go fmt ./...
go vet ./...
go test ./...
```

## 规范文档

| 文档 | 内容 |
|------|------|
| [代码规范](./docs/CODE_STYLE.md) | 命名、错误处理、格式化 |
| [协作流程](./docs/COLLABORATION.md) | 分支、PR、Conventional Commits |
| [测试规范](./docs/TESTING.md) | 测试分层与覆盖率 |
| [版本发布](./docs/RELEASE.md) | SemVer、tag、GitHub Release |
| [安全](./SECURITY.md) | 漏洞报告与红线 |
| [给 AI 的说明](./AGENTS.md) | 如何注册工具、如何改这个仓库 |

## 提交流程

1. 从最新 `main` 切分支：`feat/短描述` / `fix/短描述` / `docs/短描述`
2. 小步提交，信息遵循 Conventional Commits（可用中文 subject）
3. 推送并开 PR，等待 CI 通过后合并
4. **不要**直接在 `main` 上开发

## 反馈

- Bug：[Bug 报告模板](./.github/ISSUE_TEMPLATE/bug_report.md)
- 功能：[功能建议模板](./.github/ISSUE_TEMPLATE/feature_request.md)
- 安全：不要公开 issue，见 [SECURITY.md](./SECURITY.md)
