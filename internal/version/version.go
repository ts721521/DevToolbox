package version

import (
	"fmt"
	"regexp"
	"strings"
)

// 由构建注入；本地 `go build` 未加 ldflags 时使用这里的默认值。
var (
	Version = "1.2.0"
	Commit  = "unknown"
	Date    = "unknown"
)

var numericPrefix = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+`)

// Display is the user-facing version, always with a v prefix (e.g. v1.2.0).
func Display() string {
	v := strings.TrimSpace(Version)
	if v == "" || strings.EqualFold(v, "unknown") {
		v = "0.0.0-dev"
	}
	if v[0] >= '0' && v[0] <= '9' {
		return "v" + v
	}
	return v
}

// Tag is Display; used in file names.
func Tag() string {
	return Display()
}

// Numeric is MAJOR.MINOR.PATCH for Info.plist / Windows, without v.
func Numeric() string {
	s := strings.TrimPrefix(Display(), "v")
	if m := numericPrefix.FindString(s); m != "" {
		return m
	}
	return s
}

// BinaryName is the versioned CLI artifact, e.g. tooldock-darwin-arm64-v1.2.0
func BinaryName(goos, goarch string) string {
	ext := ""
	if goos == "windows" {
		ext = ".exe"
	}
	return fmt.Sprintf("tooldock-%s-%s-%s%s", goos, goarch, Tag(), ext)
}

// AppBundleName is the versioned macOS app folder, e.g. 工坞-v1.2.0.app
func AppBundleName() string {
	return "工坞-" + Tag() + ".app"
}

// ZipName is the versioned archive for a platform.
func ZipName(goos, goarch string) string {
	osLabel := goos
	switch goos {
	case "darwin":
		osLabel = "macOS"
	case "windows":
		osLabel = "Windows"
	case "linux":
		osLabel = "Linux"
	}
	return fmt.Sprintf("ToolDock-%s-%s-%s.zip", Tag(), osLabel, goarch)
}
