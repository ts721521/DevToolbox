# 更新日志

本文件遵循 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)，版本号遵循 [SemVer](https://semver.org/lang/zh-CN/)。

## [未发布]

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

[未发布]: https://github.com/ts721521/DevToolbox/compare/v1.1.0...HEAD
[1.1.0]: https://github.com/ts721521/DevToolbox/releases/tag/v1.1.0
[1.0.0]: https://github.com/ts721521/DevToolbox/releases/tag/v1.0.0
