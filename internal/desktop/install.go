package desktop

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/ts721521/DevToolbox/internal/platform"
	"github.com/ts721521/DevToolbox/internal/version"
)

const (
	AppDisplayName = "工坞"
	AppEnglishName = "ToolDock"
	BundleID       = "com.tooldock.app"
	BinName        = "tooldock"
	LauncherName   = "ToolDock"
	LegacyBinName  = "devtoolbox"
)

//go:embed assets/icon.png
var iconPNG []byte

//go:embed assets/macos_launcher.m
var macLauncherSrc []byte

func Install(execPath string, extra map[string][]byte) (string, error) {
	desktop, err := platform.DesktopDir()
	if err != nil {
		return "", err
	}
	var path string
	switch runtime.GOOS {
	case "darwin":
		path, err = installDarwin(desktop, execPath)
	case "windows":
		path, err = installWindows(desktop, execPath)
	default:
		path, err = installLinux(desktop, execPath)
	}
	if err != nil {
		return "", err
	}
	writeExtra(path, extra)
	return path, nil
}

func writeExtra(installed string, extra map[string][]byte) {
	if len(extra) == 0 {
		return
	}
	dir := installed
	st, err := os.Stat(installed)
	if err == nil && !st.IsDir() {
		dir = filepath.Dir(installed)
	} else if runtime.GOOS == "darwin" && strings.HasSuffix(installed, ".app") {
		dir = filepath.Join(installed, "Contents", "Resources")
	}
	_ = os.MkdirAll(dir, 0o755)
	for name, data := range extra {
		_ = os.WriteFile(filepath.Join(dir, name), data, 0o644)
	}
}

func installDarwin(desktop, execPath string) (string, error) {
	appDir, err := darwinInstallDir()
	if err != nil {
		return "", err
	}
	app := filepath.Join(appDir, AppDisplayName+".app")
	stopOldBundle(app)
	if err := writeDarwinBundle(app, execPath); err != nil {
		return "", err
	}
	refreshLaunchServices(app)
	_ = os.RemoveAll(filepath.Join(desktop, "开发工具箱.app"))
	if err := makeDesktopShortcut(app, desktop); err != nil {
		return "", fmt.Errorf("桌面快捷方式: %w", err)
	}
	return app, nil
}

func darwinInstallDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	for _, dir := range []string{"/Applications", filepath.Join(home, "Applications")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			continue
		}
		if writable(dir) {
			return dir, nil
		}
	}
	return platform.DesktopDir()
}

func writable(dir string) bool {
	f, err := os.CreateTemp(dir, ".tooldock-write-*")
	if err != nil {
		return false
	}
	name := f.Name()
	_ = f.Close()
	_ = os.Remove(name)
	return true
}

func writeDarwinBundle(app, execPath string) error {
	helpers := filepath.Join(app, "Contents", "Helpers")
	macos := filepath.Join(app, "Contents", "MacOS")
	res := filepath.Join(app, "Contents", "Resources")
	for _, d := range []string{macos, res, helpers} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}
	destBin := filepath.Join(helpers, BinName)
	if err := copyFile(execPath, destBin); err != nil {
		return err
	}
	if err := os.Chmod(destBin, 0o755); err != nil {
		return err
	}
	exeName := LauncherName
	if err := compileMacLauncher(filepath.Join(macos, LauncherName)); err != nil {
		if runtime.GOOS == "darwin" {
			return fmt.Errorf("mac launcher: %w", err)
		}
		exeName = BinName
	}
	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleName</key><string>%s</string>
    <key>CFBundleDisplayName</key><string>%s</string>
    <key>CFBundleSpokenName</key><string>%s</string>
    <key>CFBundleIdentifier</key><string>%s</string>
    <key>CFBundleVersion</key><string>%s</string>
    <key>CFBundleShortVersionString</key><string>%s</string>
    <key>CFBundlePackageType</key><string>APPL</string>
    <key>CFBundleExecutable</key><string>%s</string>
    <key>CFBundleIconFile</key><string>AppIcon</string>
    <key>CFBundleIconName</key><string>AppIcon</string>
    <key>NSHumanReadableCopyright</key><string>ToolDock %s</string>
    <key>LSApplicationCategoryType</key><string>public.app-category.developer-tools</string>
    <key>LSMinimumSystemVersion</key><string>11.0</string>
    <key>NSHighResolutionCapable</key><true/>
    <key>NSSupportsAutomaticTermination</key><false/>
</dict>
</plist>
`, AppDisplayName, AppDisplayName, AppEnglishName, BundleID, version.Numeric(), version.Numeric(), exeName, version.Display())
	if err := os.WriteFile(filepath.Join(app, "Contents", "Info.plist"), []byte(plist), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(app, "Contents", "PkgInfo"), []byte("APPL????"), 0o644); err != nil {
		return err
	}
	if err := writeIcns(filepath.Join(res, "AppIcon.icns")); err != nil {
		return fmt.Errorf("icon: %w", err)
	}
	_ = exec.Command("xattr", "-cr", app).Run()
	return nil
}

func stopOldBundle(app string) {
	if !strings.Contains(app, ".app") {
		return
	}
	_ = exec.Command("pkill", "-f", filepath.Join(app, "Contents")+"/").Run()
	time.Sleep(200 * time.Millisecond)
}

func compileMacLauncher(out string) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("not darwin")
	}
	if len(macLauncherSrc) == 0 {
		return fmt.Errorf("missing launcher source")
	}
	tmp, err := os.CreateTemp("", "tooldock-launch-*.m")
	if err != nil {
		return err
	}
	src := tmp.Name()
	_, err = tmp.Write(macLauncherSrc)
	_ = tmp.Close()
	defer os.Remove(src)
	if err != nil {
		return err
	}
	cmd := exec.Command("clang", "-Os", "-fobjc-arc", "-framework", "Cocoa", "-o", out, src)
	b, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("clang: %w (%s)", err, strings.TrimSpace(string(b)))
	}
	return os.Chmod(out, 0o755)
}

func makeDesktopShortcut(app, desktop string) error {
	for _, name := range []string{
		AppDisplayName,
		AppDisplayName + ".app",
		"开发工具箱",
		"开发工具箱.app",
	} {
		_ = os.RemoveAll(filepath.Join(desktop, name))
	}
	// 符号链接即可双击打开，且不依赖 Finder 自动化授权。
	return os.Symlink(app, filepath.Join(desktop, AppDisplayName+".app"))
}

func refreshLaunchServices(app string) {
	_ = exec.Command("touch", app).Run()
	lsregister := "/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister"
	_ = exec.Command(lsregister, "-f", app).Run()
}

func installWindows(desktop, execPath string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, "AppData", "Local", "Programs", AppEnglishName)
	_ = os.MkdirAll(dir, 0o755)
	dest := filepath.Join(dir, AppDisplayName+".exe")
	if err := copyFile(execPath, dest); err != nil {
		return "", err
	}
	desk := filepath.Join(desktop, AppDisplayName+".exe")
	_ = copyFile(execPath, desk)
	return dest, nil
}

func installLinux(desktop, execPath string) (string, error) {
	binDir, err := platform.BinDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return "", err
	}
	destBin := filepath.Join(binDir, BinName)
	if err := copyFile(execPath, destBin); err != nil {
		return "", err
	}
	_ = os.Chmod(destBin, 0o755)
	_ = os.Symlink(destBin, filepath.Join(binDir, LegacyBinName))
	iconPath := filepath.Join(binDir, BinName+".png")
	_ = os.WriteFile(iconPath, iconPNG, 0o644)
	entry := filepath.Join(desktop, BinName+".desktop")
	body := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=%s
GenericName=%s
Comment=Local launcher for development tools
Exec=%s
Icon=%s
Terminal=false
Categories=Development;
`, AppDisplayName, AppEnglishName, destBin, iconPath)
	if err := os.WriteFile(entry, []byte(body), 0o755); err != nil {
		return "", err
	}
	return entry, nil
}

func InstallCLI(execPath string) (string, error) {
	dir, err := platform.BinDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	dest := filepath.Join(dir, BinName+ext)
	if err := copyFile(execPath, dest); err != nil {
		return "", err
	}
	_ = os.Chmod(dest, 0o755)
	legacy := filepath.Join(dir, LegacyBinName+ext)
	_ = os.Remove(legacy)
	if runtime.GOOS == "windows" {
		_ = copyFile(dest, legacy)
	} else {
		_ = os.Symlink(dest, legacy)
	}
	return dest, nil
}

func copyFile(src, dst string) error {
	in, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	tmp := dst + ".tmp"
	if err := os.WriteFile(tmp, in, 0o755); err != nil {
		return err
	}
	return os.Rename(tmp, dst)
}

func writeIcns(icnsPath string) error {
	if runtime.GOOS != "darwin" {
		return os.WriteFile(strings.TrimSuffix(icnsPath, ".icns")+".png", iconPNG, 0o644)
	}
	if len(iconPNG) == 0 {
		return fmt.Errorf("missing embedded icon")
	}
	tmp, err := os.MkdirTemp("", "tooldock-icon")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	pngPath := filepath.Join(tmp, "icon.png")
	if err := os.WriteFile(pngPath, iconPNG, 0o644); err != nil {
		return err
	}
	iconset := filepath.Join(tmp, "icon.iconset")
	if err := os.MkdirAll(iconset, 0o755); err != nil {
		return err
	}
	sizes := []int{16, 32, 64, 128, 256, 512, 1024}
	for _, s := range sizes {
		out := filepath.Join(iconset, fmt.Sprintf("icon_%dx%d.png", s, s))
		if err := exec.Command("sips", "-z", fmt.Sprintf("%d", s), fmt.Sprintf("%d", s), pngPath, "--out", out).Run(); err != nil {
			return err
		}
		if s <= 512 {
			out2 := filepath.Join(iconset, fmt.Sprintf("icon_%dx%d@2x.png", s, s))
			s2 := s * 2
			if s2 <= 1024 {
				_ = exec.Command("sips", "-z", fmt.Sprintf("%d", s2), fmt.Sprintf("%d", s2), pngPath, "--out", out2).Run()
			}
		}
	}
	return exec.Command("iconutil", "-c", "icns", iconset, "-o", icnsPath).Run()
}

func SelfPath() (string, error) {
	p, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(p)
}

func OpenInstalled(path string) error {
	return platform.OpenPath(path)
}
