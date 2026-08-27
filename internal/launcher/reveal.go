package launcher

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ts721521/DevToolbox/internal/applog"
	"github.com/ts721521/DevToolbox/internal/platform"
	"github.com/ts721521/DevToolbox/internal/registry"
)

func DirToReveal(t registry.Tool) (string, error) {
	t = t.ForOS(runtime.GOOS)
	dir := strings.TrimSpace(platform.Expand(t.Workdir))
	if dir != "" {
		st, err := os.Stat(dir)
		if err != nil {
			return "", fmt.Errorf("开发目录不存在：%s", dir)
		}
		if !st.IsDir() {
			dir = filepath.Dir(dir)
		}
		return dir, nil
	}
	app := strings.TrimSpace(platform.Expand(t.AppPath))
	if app != "" {
		if _, err := os.Stat(app); err != nil {
			return "", fmt.Errorf("程序路径不存在：%s", app)
		}
		return filepath.Dir(strings.TrimSuffix(app, string(filepath.Separator))), nil
	}
	return "", fmt.Errorf("未注册开发目录 workdir（也没有 app_path）")
}

func ProgramToReveal(t registry.Tool) (string, error) {
	t = t.ForOS(runtime.GOOS)
	p := strings.TrimSpace(platform.Expand(t.AppPath))
	if p == "" {
		return "", fmt.Errorf("未注册原始程序 app_path")
	}
	if _, err := os.Stat(p); err != nil {
		return "", fmt.Errorf("程序不存在：%s", p)
	}
	return p, nil
}

func RevealDir(t registry.Tool) error {
	dir, err := DirToReveal(t)
	if err != nil {
		return err
	}
	applog.Info("reveal_dir", "id", t.ID, "path", dir)
	return platform.Reveal(dir)
}

func RevealProgram(t registry.Tool) error {
	p, err := ProgramToReveal(t)
	if err != nil {
		return err
	}
	applog.Info("reveal_app", "id", t.ID, "path", p)
	return platform.OpenPath(p)
}

func RevealGit(t registry.Tool) error {
	web := registry.GitWebURL(t.Git)
	if web == "" {
		return fmt.Errorf("未注册 git 仓库地址")
	}
	applog.Info("reveal_git", "id", t.ID, "url", web)
	return platform.OpenURL(web)
}
