# 如何把工具注册进「工坞」

完整说明见同目录 **AGENTS.md**（GitHub 仓库、安装目录、`http://127.0.0.1:17890/AGENTS.md` 三处内容相同）。

最短路径：

1. `GET http://127.0.0.1:17890/api/tabs`，把项目放进已有标签（默认：工作 / 财务 / 开发 / 其他）
2. 在项目根写 `.devtoolbox.json`
   - **必须** `workdir` = 当前项目根绝对路径（「目录」按钮 / `tooldock dir`；纯 `kind=url` 可省略）
   - 有安装好的 `.app` / `.exe` 再写 `app_path`（「程序」按钮 / `tooldock app`）
   - 必须 `group`
   - **尽量写 `git`** = `git remote get-url origin`（界面「仓库」）
   - 主程序还要先起 worker / compose / 数据库时：写 `services`（附加命令）和/或 `depends`（已注册工具的**真实** id，先 `GET /api/tools`）。点「打开」**只会**启动 JSON 里写了的东西。`health_url` 必须是本机 `http://127.0.0.1` / `localhost`。`terminal: true` 的服务不记 PID，关闭靠 `process_match` / 端口。一条 oneclick / `docker compose up` 已经带起全家时，不要再拆重复的 `services`。
3. `tooldock register --file .devtoolbox.json`（没有 CLI 时：`curl` POST 须带 `-H 'Origin: http://127.0.0.1:17890'`）
4. 不要再往桌面放单独快捷方式
5. 先 `GET /api/blocked`：垃圾桶里的项目不要再注册

启动：`tooldock open <id>`。目录：`tooldock dir <id>`。程序：`tooldock app <id>`。仓库：`tooldock git <id>`。关闭：`tooldock stop <id>`。移除：`tooldock unregister <id>`（进垃圾桶）。恢复：`tooldock restore <id>`。
支援系统：macOS + Windows。
模板：`examples/` 。
