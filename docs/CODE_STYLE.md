# 代码规范

标识符用英文，用户可见文案和文档用中文。

## 格式化

```bash
go fmt ./...
```

- Tab 缩进
- 建议行宽 ≤ 100
- 必须处理 error；测试里可用 `t.Fatal`

## 命名

| 类型 | 规则 |
|------|------|
| 包名 | 小写短词：`registry`、`launcher` |
| 导出 | `Save`、`Launch` |
| 未导出 | `portBusy` |
| 错误 | 包级 `ErrNotFound` |

## 包边界

- `internal/registry`：工具清单持久化
- `internal/launcher`：启动 / 停止 / 探活
- `internal/proc`：跨平台进程与端口
- `internal/platform`：打开 URL、终端、桌面路径
- `internal/server`：本机 HTTP API
- `internal/desktop`：安装到桌面
- `internal/docs`：把 AGENTS.md 写到配置目录
- `web/`：嵌入的界面

## 安全

- 默认只监听 `127.0.0.1`
- 示例 JSON 不得包含真实家目录、内网主机、凭据
- 关闭工具时禁止杀掉工具箱自己的进程（端口 17890）
