package platform

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func Expand(s string) string {
	if s == "" {
		return s
	}
	home, _ := os.UserHomeDir()
	s = strings.ReplaceAll(s, "${HOME}", home)
	return os.ExpandEnv(s)
}

func OpenURL(raw string) error {
	if raw == "" {
		return fmt.Errorf("empty url")
	}
	u, err := url.ParseRequestURI(raw)
	if err != nil {
		return fmt.Errorf("bad url %q: %w", raw, err)
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
	default:
		return fmt.Errorf("url scheme %q not allowed", u.Scheme)
	}
	if u.Host == "" {
		return fmt.Errorf("bad url %q", raw)
	}
	return openPathOrURL(raw)
}

func OpenPath(path string) error {
	path = Expand(path)
	if path == "" {
		return fmt.Errorf("empty path")
	}
	return openPathOrURL(path)
}

// Reveal opens a folder in the file manager, or selects a file/app in it.
func Reveal(path string) error {
	path = Expand(path)
	if path == "" {
		return fmt.Errorf("empty path")
	}
	st, err := os.Stat(path)
	if err != nil {
		return err
	}
	switch runtime.GOOS {
	case "darwin":
		isApp := strings.HasSuffix(strings.ToLower(path), ".app")
		if st.IsDir() && !isApp {
			return exec.Command("open", path).Start()
		}
		return exec.Command("open", "-R", path).Start()
	case "windows":
		if st.IsDir() {
			return exec.Command("explorer", path).Start()
		}
		return exec.Command("explorer", "/select,"+path).Start()
	default:
		target := path
		if !st.IsDir() {
			target = filepath.Dir(path)
		}
		return exec.Command("xdg-open", target).Start()
	}
}

func openPathOrURL(target string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", target)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	default:
		cmd = exec.Command("xdg-open", target)
	}
	return cmd.Start()
}

func OpenInChromeApp(raw string) error {
	chrome := findChrome()
	if chrome == "" {
		return OpenURL(raw)
	}
	cmd := exec.Command(chrome, "--app="+raw, "--window-size=1080,760")
	return cmd.Start()
}

func findChrome() string {
	switch runtime.GOOS {
	case "darwin":
		p := "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
		if fileExists(p) {
			return p
		}
		p = "/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge"
		if fileExists(p) {
			return p
		}
	case "windows":
		candidates := []string{
			filepath.Join(os.Getenv("ProgramFiles"), "Google/Chrome/Application/chrome.exe"),
			filepath.Join(os.Getenv("ProgramFiles(x86)"), "Google/Chrome/Application/chrome.exe"),
			filepath.Join(os.Getenv("LocalAppData"), "Google/Chrome/Application/chrome.exe"),
			filepath.Join(os.Getenv("ProgramFiles"), "Microsoft/Edge/Application/msedge.exe"),
		}
		for _, p := range candidates {
			if fileExists(p) {
				return p
			}
		}
	default:
		for _, p := range []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser", "microsoft-edge"} {
			if bin, err := exec.LookPath(p); err == nil {
				return bin
			}
		}
	}
	return ""
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

func RunInTerminal(workdir string, argv []string, env map[string]string) error {
	if len(argv) == 0 {
		return fmt.Errorf("empty command")
	}
	workdir = Expand(workdir)
	line := shellJoin(argv)
	if workdir != "" {
		line = "cd " + shellQuote(workdir) + " && " + line
	}
	if len(env) > 0 {
		var parts []string
		for k, v := range env {
			parts = append(parts, k+"="+shellQuote(Expand(v)))
		}
		line = strings.Join(parts, " ") + " " + line
	}
	switch runtime.GOOS {
	case "darwin":
		script := `tell application "Terminal" to do script ` + applescriptQuote(line)
		return exec.Command("osascript", "-e", script).Start()
	case "windows":
		return exec.Command("cmd", "/c", "start", "cmd", "/k", line).Start()
	default:
		for _, term := range []string{"x-terminal-emulator", "gnome-terminal", "konsole", "xfce4-terminal", "xterm"} {
			if bin, err := exec.LookPath(term); err == nil {
				if strings.Contains(term, "gnome-terminal") {
					return exec.Command(bin, "--", "bash", "-lc", line).Start()
				}
				return exec.Command(bin, "-e", "bash", "-lc", line).Start()
			}
		}
		return fmt.Errorf("no terminal emulator found")
	}
}

func StartDetached(workdir string, argv []string, env map[string]string) (int, error) {
	if len(argv) == 0 {
		return 0, fmt.Errorf("empty command")
	}
	expanded := make([]string, len(argv))
	for i, a := range argv {
		expanded[i] = Expand(a)
	}
	cmd := exec.Command(expanded[0], expanded[1:]...)
	if workdir != "" {
		cmd.Dir = Expand(workdir)
	}
	if len(env) > 0 {
		cmd.Env = os.Environ()
		for k, v := range env {
			cmd.Env = append(cmd.Env, k+"="+Expand(v))
		}
	}
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.SysProcAttr = DetachAttr()
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	pid := cmd.Process.Pid
	go func() { _ = cmd.Wait() }()
	return pid, nil
}

func shellJoin(argv []string) string {
	parts := make([]string, len(argv))
	for i, a := range argv {
		parts[i] = shellQuote(Expand(a))
	}
	return strings.Join(parts, " ")
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

func applescriptQuote(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
}

func DesktopDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	desktop := filepath.Join(home, "Desktop")
	if st, err := os.Stat(desktop); err == nil && st.IsDir() {
		return desktop, nil
	}
	// Some locales / OneDrive
	if runtime.GOOS == "windows" {
		if p := os.Getenv("USERPROFILE"); p != "" {
			d := filepath.Join(p, "Desktop")
			if st, err := os.Stat(d); err == nil && st.IsDir() {
				return d, nil
			}
		}
	}
	return desktop, nil
}

func BinDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if runtime.GOOS == "windows" {
		return filepath.Join(home, "bin"), nil
	}
	return filepath.Join(home, "bin"), nil
}
