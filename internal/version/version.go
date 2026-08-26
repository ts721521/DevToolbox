package version

// 由构建注入；本地 `go build` 时使用这里的默认值。
var (
	Version = "1.0.0"
	Commit  = "unknown"
	Date    = "unknown"
)
