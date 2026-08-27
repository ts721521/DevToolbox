package desktop

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ts721521/DevToolbox/internal/version"
)

// Pack writes versioned binaries (and on macOS a versioned .app + zip) into outDir.
func Pack(outDir, execPath string) ([]string, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, err
	}
	written, err := packBinaries(outDir, execPath)
	if err != nil {
		return nil, err
	}

	if runtime.GOOS == "darwin" {
		app := filepath.Join(outDir, version.AppBundleName())
		if err := writeDarwinBundle(app, execPath); err != nil {
			return written, err
		}
		written = append(written, app)
		zipPath := filepath.Join(outDir, version.ZipName("darwin", runtime.GOARCH))
		if err := zipDir(app, zipPath); err != nil {
			return written, fmt.Errorf("zip app: %w", err)
		}
		written = append(written, zipPath)
	}
	if runtime.GOOS == "windows" {
		win := filepath.Join(outDir, "ToolDock-"+version.Tag()+"-windows-"+runtime.GOARCH+".exe")
		if err := copyFile(execPath, win); err != nil {
			return written, err
		}
		written = append(written, win)
	}
	sum, err := writeChecksums(outDir, written)
	if err != nil {
		return written, err
	}
	if sum != "" {
		written = append(written, sum)
	}
	return written, nil
}

func packBinaries(outDir, execPath string) ([]string, error) {
	binName := version.BinaryName(runtime.GOOS, runtime.GOARCH)
	binPath := filepath.Join(outDir, binName)
	if err := copyFile(execPath, binPath); err != nil {
		return nil, err
	}
	_ = os.Chmod(binPath, 0o755)
	written := []string{binPath}

	legacy := filepath.Join(outDir, "devtoolbox-"+runtime.GOOS+"-"+runtime.GOARCH+"-"+version.Tag())
	if runtime.GOOS == "windows" {
		legacy += ".exe"
	}
	if err := copyFile(binPath, legacy); err != nil {
		return written, err
	}
	return append(written, legacy), nil
}

func zipDir(src, zipPath string) error {
	_ = os.Remove(zipPath)
	if _, err := exec.LookPath("ditto"); err == nil {
		return exec.Command("ditto", "-c", "-k", "--keepParent", src, zipPath).Run()
	}
	parent := filepath.Dir(src)
	base := filepath.Base(src)
	cmd := exec.Command("zip", "-r", "-q", zipPath, base)
	cmd.Dir = parent
	return cmd.Run()
}

func writeChecksums(outDir string, files []string) (string, error) {
	var b strings.Builder
	for _, f := range files {
		st, err := os.Stat(f)
		if err != nil || st.IsDir() {
			continue
		}
		sum, err := fileSHA256(f)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "%s  %s\n", sum, filepath.Base(f))
	}
	if b.Len() == 0 {
		return "", nil
	}
	path := filepath.Join(outDir, "SHA256SUMS-"+version.Tag()+".txt")
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
