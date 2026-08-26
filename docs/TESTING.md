# 测试规范

## 命令

```bash
go test ./...
go test -race ./...
go test -coverprofile=coverage.out ./...
```

## 分层

| 层 | 位置 | 测什么 |
|----|------|--------|
| 单元 | `internal/*/*_test.go` | 注册表、端口解析、平台归一化 |
| 构建 | CI `go build` | macOS / Windows / Linux 交叉编译 |

新的导出函数或会出错的解析逻辑，优先用表驱动测试。

HTTP 启动/杀进程涉及真实 OS，放在单元测试里要用假数据或 `httptest`，不要依赖开发者本机已开的服务。

## 覆盖率

不为百分比而测。注册表校验、端口解析、平台别名必须有断言。
