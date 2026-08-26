package desktop

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ts721521/DevToolbox/internal/platform"
)

const AppDisplayName = "开发工具箱"

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
	app := filepath.Join(desktop, AppDisplayName+".app")
	macos := filepath.Join(app, "Contents", "MacOS")
	res := filepath.Join(app, "Contents", "Resources")
	for _, d := range []string{macos, res} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return "", err
		}
	}
	destBin := filepath.Join(macos, "devtoolbox")
	if err := copyFile(execPath, destBin); err != nil {
		return "", err
	}
	if err := os.Chmod(destBin, 0o755); err != nil {
		return "", err
	}
	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleName</key><string>%s</string>
    <key>CFBundleDisplayName</key><string>%s</string>
    <key>CFBundleIdentifier</key><string>com.devtoolbox.app</string>
    <key>CFBundleVersion</key><string>1.0.0</string>
    <key>CFBundleShortVersionString</key><string>1.0.0</string>
    <key>CFBundlePackageType</key><string>APPL</string>
    <key>CFBundleExecutable</key><string>devtoolbox</string>
    <key>LSMinimumSystemVersion</key><string>10.15</string>
    <key>NSHighResolutionCapable</key><true/>
    <key>CFBundleIconFile</key><string>AppIcon</string>
</dict>
</plist>
`, AppDisplayName, AppDisplayName)
	if err := os.WriteFile(filepath.Join(app, "Contents", "Info.plist"), []byte(plist), 0o644); err != nil {
		return "", err
	}
	if err := writeIcns(filepath.Join(res, "AppIcon.icns")); err != nil {
		// icon is optional
		_ = err
	}
	return app, nil
}

func installWindows(desktop, execPath string) (string, error) {
	dest := filepath.Join(desktop, AppDisplayName+".exe")
	if err := copyFile(execPath, dest); err != nil {
		return "", err
	}
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
	destBin := filepath.Join(binDir, "devtoolbox")
	if err := copyFile(execPath, destBin); err != nil {
		return "", err
	}
	_ = os.Chmod(destBin, 0o755)
	entry := filepath.Join(desktop, "devtoolbox.desktop")
	body := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=%s
Comment=Central launcher for local dev tools
Exec=%s
Terminal=false
Categories=Development;
`, AppDisplayName, destBin)
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
	name := "devtoolbox"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	dest := filepath.Join(dir, name)
	if err := copyFile(execPath, dest); err != nil {
		return "", err
	}
	_ = os.Chmod(dest, 0o755)
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
		return nil
	}
	tmp, err := os.MkdirTemp("", "devtoolbox-icon")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	pngPath := filepath.Join(tmp, "icon.png")
	if err := writePNG(pngPath, 1024); err != nil {
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

func writePNG(path string, size int) error {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	bg := color.RGBA{10, 10, 18, 255}
	accent := color.RGBA{59, 130, 246, 255}
	purple := color.RGBA{139, 92, 246, 255}
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.Set(x, y, bg)
		}
	}
	margin := size / 8
	for y := margin; y < size-margin; y++ {
		for x := margin; x < size-margin; x++ {
			t := float64(x-margin) / float64(size-2*margin)
			c := color.RGBA{
				R: uint8(float64(accent.R)*(1-t) + float64(purple.R)*t),
				G: uint8(float64(accent.G)*(1-t) + float64(purple.G)*t),
				B: uint8(float64(accent.B)*(1-t) + float64(purple.B)*t),
				A: 255,
			}
			img.Set(x, y, c)
		}
	}
	inner := size / 4
	for y := inner; y < size-inner; y++ {
		for x := inner; x < size-inner; x++ {
			if (x+y)%(size/16) < size/48 {
				continue
			}
			img.Set(x, y, bg)
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
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

func Quote(s string) string {
	return strings.TrimSpace(s)
}
