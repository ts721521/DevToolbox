# 更新日志

本文件遵循 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)，版本号遵循 [SemVer](https://semver.org/lang/zh-CN/)。

## [未发布]

### 变更
- PR 合入 `main` 由 GitHub ruleset 强制走 PR + 汇总检查 `CI`；用户同意后任意 AI 用 `gh pr merge --auto` 合并

## [1.2.0] - 2026-08-27

### 新增
- 打开时连带启动依赖：`services`（附加进程）和 `depends`（已注册的其他工具）。已在跑的会跳过；关闭只停本工具及其 `services`，不连带关 `depends`
- 发给 AI 的注册协议强制要求写清打开时要先起的 `services`/`depends`；没写进 JSON 的后台不会自动启动
- 注册成功时若没有 `services`/`depends`，CLI 和 API 会提示补写
- 每个工具可填 `git` 仓库地址；界面「仓库」/ `tooldock git` 打开网页
- 工坞界面重做：石墨底、酸性点缀、更少边框噪音
- 界面轮询不再整页重绘；数据没变时卡片不会反复闪入场动画
- 界面、CLI、`/api/health` 显示版本号（`vX.Y.Z`）
- 发布与本地封装产物文件名带版本：`tooldock-<os>-<arch>-vX.Y.Z`、macOS `工坞-vX.Y.Z.app` / `ToolDock-vX.Y.Z-macOS-<arch>.zip`
- Mac `Info.plist` 写入短版本号，可在「关于」里看到
- 移除的工具进入本机垃圾桶并屏蔽同 id / 目录 / 仓库，扫描和 AI 不能再注册进来；可恢复或取消屏蔽
- 工坞只跑一份：再打开会唤起已有界面；安装了新版本则顶掉旧进程
- 从「应用程序」再次点开工坞会弹出界面；程序坞显示图标，不再空跳

### 修复
- 关闭不再按 URL 默认的 80/443 杀进程；只杀本机显式端口
- 依赖已在跑时仍会补齐它的 `services`
- 附加服务中途失败会停掉已拉起的 sidecar，避免孤儿进程
- 工具 ID 禁止路径穿越；`url` 只允许 http(s)
- 浏览器跨站 Origin 的 POST/DELETE 返回 403
- 无 Origin 的改写请求须带 `X-ToolDock-Token`（界面从 `/api/system` 读取）
- `health_url` 只允许本机 http(s)，工坞不再向外网做健康检查
- 附加服务健康检查超时会失败并回滚
- 健康检查不跟随跳出 loopback 的 HTTP 重定向；仅 2xx 算就绪
- 附加服务启动失败后清空残留 `service_pids`
- `scripts/install.sh` 校验 `SHA256SUMS-vX.Y.Z.txt`
- macOS 发布构建不再用 `-s` 剥离，避免 dyld 因缺少 LC_UUID 直接 Abort
- 打开后立刻刷新状态；关闭失败不再把按钮卡死
- CLI `stop` 不会误杀工坞进程；`:17890` 被其他程序占用时不会误打开
- 从系统「应用程序」点开工坞时能唤起界面（Cocoa 启动器）
- 纯链接工具点关闭不会按端口杀进程，避免误停依赖或其他本机服务

## [1.1.0] - 2026-08-26

### 新增
- 产品名改为 **工坞 / ToolDock**（Mac 安装到 `/Applications/工坞.app`，CLI `tooldock`，兼容 `devtoolbox`）
- 「发给 AI」：复制本机注册说明书地址与提示词；运行中的 `/for-ai.md`、`/ai` 给其他 AI 自注册
- 自定义标签页：拖到标签、单选/多选批量移动；默认 工作 / 财务 / 开发 / 其他
- 运行日志（文件 + 界面 + `tooldock logs`）
- 打开开发目录 / 原始程序（界面「目录」「程序」，CLI `tooldock dir` / `tooldock app`）
- 其他 AI 注册时必须先读 `/api/tabs` 自行分类，并填写 `workdir`（`kind=url` 除外）及可选 `app_path`
- `web` / `command` / `app` 注册时强制 `workdir`，与文档和 schema 对齐

### 修复
- CI：Windows 覆盖率参数、macOS race 与 staticcheck HTTP 状态码
- 删除标签后 `moveTo` 目标被 `SaveTabs` 冲掉
- 同名重命名标签会误删空标签
- HTTP panic recover 可能二次写响应或把 500 记成 200

## [1.0.0] - 2026-08-26

### 新增
- 跨平台本地工具启动器（macOS / Windows，Go 单二进制）
- 桌面安装：`devtoolbox install-desktop` / 命令行：`install-cli`
- 工具注册：JSON 文件、CLI、HTTP API
- 每张卡片显示 macOS / Windows 支援
- 一键关闭工具及其后台进程（端口 + 进程匹配）
- 供其他 AI 阅读的 `AGENTS.md`（源码目录、安装目录、运行中的 `/AGENTS.md`）

[未发布]: https://github.com/ts721521/DevToolbox/compare/v1.2.0...HEAD
[1.2.0]: https://github.com/ts721521/DevToolbox/releases/tag/v1.2.0
[1.1.0]: https://github.com/ts721521/DevToolbox/releases/tag/v1.1.0
[1.0.0]: https://github.com/ts721521/DevToolbox/releases/tag/v1.0.0
