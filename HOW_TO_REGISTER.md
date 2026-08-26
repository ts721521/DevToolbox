# 如何把工具注册进「工坞」

完整说明见同目录 **AGENTS.md**（GitHub 仓库、安装目录、`http://127.0.0.1:17890/AGENTS.md` 三处内容相同）。

最短路径：

1. `GET http://127.0.0.1:17890/api/tabs`，把项目放进已有标签（默认：工作 / 财务 / 开发 / 其他）
2. 在项目根写 `.devtoolbox.json`
   - **必须** `workdir` = 当前项目根绝对路径（「目录」按钮 / `tooldock dir`；纯 `kind=url` 可省略）
   - 有安装好的 `.app` / `.exe` 再写 `app_path`（「程序」按钮 / `tooldock app`）
   - 必须 `group`
3. `tooldock register --file .devtoolbox.json`
4. 不要再往桌面放单独快捷方式

启动：`tooldock open <id>`。目录：`tooldock dir <id>`。程序：`tooldock app <id>`。关闭：`tooldock stop <id>`。
支援系统：macOS + Windows。
模板：`examples/` 。
