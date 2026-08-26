# AGENTS.md — 开发工具箱 (DevToolbox)

> 给 **AI 编程助手** 和人类：这是本机所有开发工具的统一启动器。
> 读到本文件（**源码目录**、**安装目录** 或 `http://127.0.0.1:17890/AGENTS.md`）就应按下面方式注册新工具，**不要再往桌面生成单独快捷方式**。
>
> 改本仓库代码时，同时遵守 [CONTRIBUTING.md](./CONTRIBUTING.md) 与 [docs/COLLABORATION.md](./docs/COLLABORATION.md)：功能走 `feat/` 分支 + PR，版本走 SemVer tag。

## 项目快照

| 项 | 值 |
|----|----|
| 名称 | 工坞 / ToolDock |
| 仓库 | https://github.com/ts721521/DevToolbox |
| 技术栈 | Go 1.22+ · 标准库 HTTP · 嵌入式 Web UI |
| 支援系统 | **macOS + Windows**（Linux 可编译） |
| 本机 UI | `http://127.0.0.1:17890` |
| CLI | `tooldock`（兼容别名 `devtoolbox`） |
| 许可证 | MIT |

## 注册新工具（必须这样做）

用户说「加个快捷方式 / 方便打开某个工具」时：

1. 在**那个项目根目录**写 `.devtoolbox.json`（schema 见 `tool.schema.json` / `examples/`）。
2. 执行：

```bash
tooldock register --file .devtoolbox.json
```

CLI 不在 PATH 时：

- macOS: `~/bin/tooldock`（兼容 `devtoolbox`）或 `/Applications/工坞.app`
- Windows: `%USERPROFILE%\bin\tooldock.exe` 或 `%LOCALAPPDATA%\Programs\ToolDock\`

3. **不要**再创建 `.command` / `.lnk` 到桌面。

不启动界面也可以：把 JSON 复制到

| 系统 | 目录 |
|------|------|
| macOS | `~/Library/Application Support/DevToolbox/tools/<id>.json` |
| Windows | `%AppData%\DevToolbox\tools\<id>.json` |
| Linux | `~/.config/DevToolbox/tools/<id>.json` |

或 `POST http://127.0.0.1:17890/api/tools`。

安装目录里也有本文件：

- macOS：`/Applications/工坞.app/Contents/Resources/AGENTS.md`
- Windows：exe 旁边的 `AGENTS.md`
- 配置根：`…/DevToolbox/AGENTS.md`

## JSON 要点

- `id`：`[a-zA-Z0-9][a-zA-Z0-9._-]*`
- `kind`：`web` | `command` | `app` | `url`
- `group`：标签页名称。注册前先 `GET /api/tabs`，优先用已有标签（默认：**工作 / 财务 / 开发 / 其他**），不要自己发明一堆分类
- **`workdir` 必填**（`kind=url` 除外）：当前项目根绝对路径。用户点「目录」或 `tooldock dir <id>` 会打开这里
- 已安装的桌面程序再写 `app_path`（`.app` / `.exe`）。「程序」/ `tooldock app <id>` 打开它。`kind=app` 时 **workdir + app_path 都要写**
- `platforms`：`darwin` / `windows`（默认两者都有）
- Windows 覆盖：`command_windows`、`workdir_windows`、`app_path_windows`
- `process_match`：关闭时用来找后台进程
- 示例里**禁止**写真实家目录或凭据，用 `/path/to/project`

打开 / 关闭：

```bash
tooldock open <id>
tooldock dir <id>
tooldock app <id>
tooldock stop <id>
tooldock list
tooldock version
tooldock logs
```

HTTP：`POST /api/tools/{id}/launch` · `POST /api/tools/{id}/dir` · `POST /api/tools/{id}/app` · `POST /api/tools/{id}/stop` · `GET /api/system` · `GET /api/tabs` · `POST /api/tools/move` · `GET /api/logs`

## 排障日志

启动、关闭、注册、HTTP 4xx/5xx 都会写进文本日志（同时打 stderr）。界面右上角「日志」可看最近记录。

| 系统 | 文件 |
|------|------|
| macOS | `~/Library/Application Support/DevToolbox/logs/tooldock.log` |
| Windows | `%AppData%\DevToolbox\logs\tooldock.log` |
| Linux | `~/.config/DevToolbox/logs/tooldock.log` |

超过约 2MB 会滚到 `tooldock.log.1`。查 bug 时先读这个文件。

## 改本仓库

```bash
go test ./...
go build -o devtoolbox .
```

- 分支：`feat/` `fix/` `docs/` `chore/`，PR 合入 `main`
- Commit：Conventional Commits（subject 可用中文）
- 发版：`./scripts/publish.sh vX.Y.Z`（详见 docs/RELEASE.md）
- 用户可见变化必须改 `CHANGELOG.md`

## 黄金法则

1. 不要把凭据或本机绝对路径提交到 git。
2. 不要在 `main` 上直接开发功能。
3. 关闭进程时不得杀掉工具箱自己（`:17890`）。
4. 先读再改，每个 Go 改动要能 `go test` / `go vet`。
