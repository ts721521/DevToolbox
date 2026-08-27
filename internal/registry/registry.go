package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

const AppName = "DevToolbox"

var idPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

var (
	ErrNotFound = errors.New("tool not found")
	ErrInvalid  = errors.New("invalid tool")
)

// Tool is a launchable entry in the toolbox.
type Tool struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Description    string            `json:"description,omitempty"`
	Group          string            `json:"group,omitempty"`
	Kind           string            `json:"kind"` // web | command | app | url
	Workdir        string            `json:"workdir,omitempty"`
	Command        []string          `json:"command,omitempty"`
	URL            string            `json:"url,omitempty"`
	HealthURL      string            `json:"health_url,omitempty"`
	HealthContains string            `json:"health_contains,omitempty"`
	AppPath        string            `json:"app_path,omitempty"`
	Terminal       bool              `json:"terminal,omitempty"`
	Env            map[string]string `json:"env,omitempty"`
	Tags           []string          `json:"tags,omitempty"`
	Platforms      []string          `json:"platforms,omitempty"` // darwin, windows, linux
	ProcessMatch   string            `json:"process_match,omitempty"`
	StopCommand    []string          `json:"stop_command,omitempty"`
	CommandDarwin  []string          `json:"command_darwin,omitempty"`
	CommandWindows []string          `json:"command_windows,omitempty"`
	WorkdirDarwin  string            `json:"workdir_darwin,omitempty"`
	WorkdirWindows string            `json:"workdir_windows,omitempty"`
	AppPathDarwin  string            `json:"app_path_darwin,omitempty"`
	AppPathWindows string            `json:"app_path_windows,omitempty"`
	// Git is the project remote (https or git@). Shown as「仓库」in the UI.
	Git string `json:"git,omitempty"`
	// Depends are other registered tool IDs started before this tool (if not already running).
	Depends []string `json:"depends,omitempty"`
	// Services are extra processes started with this tool (workers, compose, local DB, …).
	Services []Service `json:"services,omitempty"`
}

const maxServices = 12

// Service is a sidecar process launched with a tool.
type Service struct {
	Name           string            `json:"name,omitempty"`
	Command        []string          `json:"command,omitempty"`
	CommandDarwin  []string          `json:"command_darwin,omitempty"`
	CommandWindows []string          `json:"command_windows,omitempty"`
	Workdir        string            `json:"workdir,omitempty"`
	WorkdirDarwin  string            `json:"workdir_darwin,omitempty"`
	WorkdirWindows string            `json:"workdir_windows,omitempty"`
	Env            map[string]string `json:"env,omitempty"`
	Terminal       bool              `json:"terminal,omitempty"`
	HealthURL      string            `json:"health_url,omitempty"`
	HealthContains string            `json:"health_contains,omitempty"`
	ProcessMatch   string            `json:"process_match,omitempty"`
	WaitMS         int               `json:"wait_ms,omitempty"`
}

func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, AppName, "tools"), nil
}

func RootDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, AppName), nil
}

func EnsureDir() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return dir, os.MkdirAll(dir, 0o755)
}

func pathFor(id string) (string, error) {
	if !idPattern.MatchString(id) {
		return "", fmt.Errorf("%w: id %q", ErrInvalid, id)
	}
	dir, err := EnsureDir()
	if err != nil {
		return "", err
	}
	p := filepath.Join(dir, id+".json")
	rel, err := filepath.Rel(dir, p)
	if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return "", fmt.Errorf("%w: id %q", ErrInvalid, id)
	}
	return p, nil
}

func Validate(t Tool) error {
	if !idPattern.MatchString(t.ID) {
		return fmt.Errorf("%w: id %q", ErrInvalid, t.ID)
	}
	if strings.TrimSpace(t.Name) == "" {
		return fmt.Errorf("%w: name required", ErrInvalid)
	}
	if cleanTabName(t.Group) == TrashTab {
		return fmt.Errorf("%w: group cannot be %s", ErrInvalid, TrashTab)
	}
	kind := strings.ToLower(strings.TrimSpace(t.Kind))
	if kind == "" {
		kind = inferKind(t)
	}
	switch kind {
	case "web", "command", "app", "url":
	default:
		return fmt.Errorf("%w: kind %q", ErrInvalid, t.Kind)
	}
	if kind == "web" && t.URL == "" {
		return fmt.Errorf("%w: web tool needs url", ErrInvalid)
	}
	if kind == "command" && len(t.Command) == 0 && len(t.CommandDarwin) == 0 && len(t.CommandWindows) == 0 {
		return fmt.Errorf("%w: command tool needs command", ErrInvalid)
	}
	if kind == "app" && strings.TrimSpace(t.AppPath) == "" && strings.TrimSpace(t.AppPathDarwin) == "" && strings.TrimSpace(t.AppPathWindows) == "" {
		return fmt.Errorf("%w: app tool needs app_path", ErrInvalid)
	}
	if kind == "url" && t.URL == "" {
		return fmt.Errorf("%w: url tool needs url", ErrInvalid)
	}
	if err := CheckHTTPURL(t.URL, "url"); err != nil {
		return err
	}
	if err := CheckLocalHTTPURL(t.HealthURL, "health_url"); err != nil {
		return err
	}
	if kind != "url" && strings.TrimSpace(t.Workdir) == "" {
		return fmt.Errorf("%w: %s tool needs workdir", ErrInvalid, kind)
	}
	if g := strings.TrimSpace(t.Git); g != "" && GitWebURL(g) == "" {
		return fmt.Errorf("%w: git must be an http(s) or git@ remote", ErrInvalid)
	}
	if len(t.Services) > maxServices {
		return fmt.Errorf("%w: at most %d services", ErrInvalid, maxServices)
	}
	for i, s := range t.Services {
		cmd := s.Command
		if len(cmd) == 0 {
			cmd = s.CommandDarwin
		}
		if len(cmd) == 0 {
			cmd = s.CommandWindows
		}
		if len(cmd) == 0 {
			return fmt.Errorf("%w: services[%d] needs command", ErrInvalid, i)
		}
		if err := CheckLocalHTTPURL(s.HealthURL, "services.health_url"); err != nil {
			return err
		}
	}
	for _, id := range t.Depends {
		if !idPattern.MatchString(id) {
			return fmt.Errorf("%w: depends id %q", ErrInvalid, id)
		}
		if id == t.ID {
			return fmt.Errorf("%w: tool cannot depend on itself", ErrInvalid)
		}
	}
	return nil
}

// ExtrasHint reminds registrars that Open only starts what the JSON lists.
func ExtrasHint(t Tool) string {
	if len(t.Services) > 0 || len(t.Depends) > 0 {
		return ""
	}
	return "未写 services/depends。点「打开」只会跑本条 command / app / url；同一仓库的 worker、compose、数据库或另一张已注册卡片都不会自动起。需要的话补 JSON 再注册。"
}

func inferKind(t Tool) string {
	switch {
	case t.URL != "" && (len(t.Command) > 0 || t.HealthURL != ""):
		return "web"
	case strings.TrimSpace(t.AppPath) != "":
		return "app"
	case t.URL != "":
		return "url"
	default:
		return "command"
	}
}

func Normalize(t Tool) Tool {
	t.ID = strings.TrimSpace(t.ID)
	t.Name = strings.TrimSpace(t.Name)
	t.Kind = strings.ToLower(strings.TrimSpace(t.Kind))
	if t.Kind == "" {
		t.Kind = inferKind(t)
	}
	t.Platforms = NormalizePlatforms(t.Platforms)
	t.Group = cleanTabName(t.Group)
	if t.Group == "" {
		t.Group = DefaultOther
	}
	t.Depends = uniqueIDs(t.Depends)
	t.Git = ScrubGit(t.Git)
	return t
}

func uniqueIDs(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, id := range in {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func CanonicalOS(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "darwin", "mac", "macos", "osx":
		return "darwin"
	case "windows", "win", "win32", "win64":
		return "windows"
	case "linux":
		return "linux"
	default:
		return strings.ToLower(strings.TrimSpace(s))
	}
}

func NormalizePlatforms(in []string) []string {
	if len(in) == 0 {
		return []string{"darwin", "windows"}
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, p := range in {
		c := CanonicalOS(p)
		if c == "" {
			continue
		}
		if _, ok := seen[c]; ok {
			continue
		}
		seen[c] = struct{}{}
		out = append(out, c)
	}
	if len(out) == 0 {
		return []string{"darwin", "windows"}
	}
	return out
}

func (t Tool) Supports(goos string) bool {
	want := CanonicalOS(goos)
	for _, p := range t.Platforms {
		if p == want {
			return true
		}
	}
	return len(t.Platforms) == 0
}

func PlatformLabel(goos string) string {
	switch CanonicalOS(goos) {
	case "darwin":
		return "macOS"
	case "windows":
		return "Windows"
	case "linux":
		return "Linux"
	default:
		return goos
	}
}

func (t Tool) ForOS(goos string) Tool {
	switch CanonicalOS(goos) {
	case "windows":
		if len(t.CommandWindows) > 0 {
			t.Command = t.CommandWindows
		}
		if t.WorkdirWindows != "" {
			t.Workdir = t.WorkdirWindows
		}
		if t.AppPathWindows != "" {
			t.AppPath = t.AppPathWindows
		}
	case "darwin":
		if len(t.CommandDarwin) > 0 {
			t.Command = t.CommandDarwin
		}
		if t.WorkdirDarwin != "" {
			t.Workdir = t.WorkdirDarwin
		}
		if t.AppPathDarwin != "" {
			t.AppPath = t.AppPathDarwin
		}
	}
	for i := range t.Services {
		t.Services[i] = t.Services[i].ForOS(goos)
	}
	return t
}

func (s Service) ForOS(goos string) Service {
	switch CanonicalOS(goos) {
	case "windows":
		if len(s.CommandWindows) > 0 {
			s.Command = s.CommandWindows
		}
		if s.WorkdirWindows != "" {
			s.Workdir = s.WorkdirWindows
		}
	case "darwin":
		if len(s.CommandDarwin) > 0 {
			s.Command = s.CommandDarwin
		}
		if s.WorkdirDarwin != "" {
			s.Workdir = s.WorkdirDarwin
		}
	}
	return s
}

func Save(t Tool) error {
	return saveTool(t, true)
}

func saveTool(t Tool, blockNew bool) error {
	t = Normalize(t)
	if err := Validate(t); err != nil {
		return err
	}
	p, err := pathFor(t.ID)
	if err != nil {
		return err
	}
	_, statErr := os.Stat(p)
	exists := statErr == nil
	if blockNew && !exists {
		if err := CheckBlocked(t); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(p, append(data, '\n'), 0o644); err != nil {
		return err
	}
	return EnsureTab(t.Group)
}

func Get(id string) (Tool, error) {
	p, err := pathFor(id)
	if err != nil {
		return Tool{}, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return Tool{}, fmt.Errorf("%w: %s", ErrNotFound, id)
		}
		return Tool{}, err
	}
	var t Tool
	if err := json.Unmarshal(data, &t); err != nil {
		return Tool{}, err
	}
	t = Normalize(t)
	if err := Validate(t); err != nil {
		return Tool{}, err
	}
	return t, nil
}

func Remove(id string) error {
	p, err := pathFor(id)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %s", ErrNotFound, id)
		}
		return err
	}
	var t Tool
	if json.Unmarshal(data, &t) != nil || strings.TrimSpace(t.ID) == "" {
		t.ID = id
	}
	t = Normalize(t)
	if err := os.Remove(p); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %s", ErrNotFound, id)
		}
		return err
	}
	if err := Bury(t); err != nil {
		_ = os.WriteFile(p, data, 0o644)
		return err
	}
	return nil
}

func List() ([]Tool, error) {
	dir, err := EnsureDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	tools := make([]Tool, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var t Tool
		if err := json.Unmarshal(data, &t); err != nil {
			continue
		}
		t = Normalize(t)
		if Validate(t) != nil {
			continue
		}
		tools = append(tools, t)
	}
	sort.Slice(tools, func(i, j int) bool {
		gi, gj := tools[i].Group, tools[j].Group
		if gi != gj {
			return gi < gj
		}
		return lessFold(tools[i].Name, tools[j].Name)
	})
	return tools, nil
}

func lessFold(a, b string) bool {
	for _, r := range a {
		_ = unicode.ToLower(r)
		break
	}
	return strings.ToLower(a) < strings.ToLower(b)
}
