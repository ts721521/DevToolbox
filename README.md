# 工坞 (ToolDock)

[![CI](https://github.com/ts721521/DevToolbox/actions/workflows/ci.yml/badge.svg)](https://github.com/ts721521/DevToolbox/actions/workflows/ci.yml)
[![Release](https://github.com/ts721521/DevToolbox/actions/workflows/release.yml/badge.svg)](https://github.com/ts721521/DevToolbox/actions/workflows/release.yml)
[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

跨平台本地开发工具启动器（**macOS + Windows**）。产品名 **工坞**，英文 **ToolDock**。所有开发工具停靠在一个入口；新项目写一份 `.devtoolbox.json` 即可注册。

## 文档导航

| 文档 | 说明 |
|------|------|
| [AGENTS.md](./AGENTS.md) | **给 AI / 新工具作者**：如何注册、打开、关闭 |
| [CONTRIBUTING.md](./CONTRIBUTING.md) | 如何参与开发 |
| [代码规范](./docs/CODE_STYLE.md) | 命名、错误处理 |
| [协作流程](./docs/COLLABORATION.md) | 分支、PR、Conventional Commits |
| [测试规范](./docs/TESTING.md) | 测试分层 |
| [版本发布](./docs/RELEASE.md) | SemVer、tag、GitHub Release |
| [CHANGELOG.md](./CHANGELOG.md) | 版本变更 |
| [SECURITY.md](./SECURITY.md) | 安全与漏洞报告 |

## 安装

### 从 Release 下载

到 [Releases](https://github.com/ts721521/DevToolbox/releases) 下载对应平台文件：

| 平台 | 文件 |
|------|------|
| macOS Apple Silicon | `devtoolbox-darwin-arm64-vX.Y.Z` |
| macOS Intel | `devtoolbox-darwin-amd64-vX.Y.Z` |
| Windows x64 | `devtoolbox-windows-amd64-vX.Y.Z.exe` |
| Windows ARM | `devtoolbox-windows-arm64-vX.Y.Z.exe` |
| Linux x64 | `devtoolbox-linux-amd64-vX.Y.Z` |

```bash
# macOS 去掉隔离属性后安装
xattr -d com.apple.quarantine devtoolbox-darwin-arm64-v1.0.0
chmod +x devtoolbox-darwin-arm64-v1.0.0
./devtoolbox-darwin-arm64-v1.0.0 install-cli
./devtoolbox-darwin-arm64-v1.0.0 install-desktop
# 正式安装位置：/Applications/工坞.app ，桌面为快捷方式

```

已装 `gh` 时：

```bash
./scripts/install.sh
```

### 从源码

```bash
git clone https://github.com/ts721521/DevToolbox.git
cd DevToolbox
go test ./...
go build -o devtoolbox .
./devtoolbox install-desktop
./devtoolbox install-cli
./devtoolbox
```

界面：`http://127.0.0.1:17890`（仅本机）。

## 注册工具

```bash
devtoolbox register --file .devtoolbox.json
```

模板见 [`examples/`](./examples/)。完整协议见 [AGENTS.md](./AGENTS.md)。

关闭后台：`devtoolbox stop <id>` 或界面上的「关闭」。
打开开发目录：`tooldock dir <id>`。打开原始程序：`tooldock app <id>`。

## 发布新版本（维护者）

```bash
./scripts/publish.sh           # 推断下一 PATCH
./scripts/publish.sh v1.1.0    # 指定版本
./scripts/publish.sh v1.1.0 -y
```

推送 `v*` tag 后，GitHub Actions 会交叉编译并创建 Release。

## License

[MIT](LICENSE)
