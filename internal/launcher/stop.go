package launcher

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/ts721521/DevToolbox/internal/applog"
	"github.com/ts721521/DevToolbox/internal/platform"
	"github.com/ts721521/DevToolbox/internal/proc"
	"github.com/ts721521/DevToolbox/internal/registry"
)

type RunRecord struct {
	ID        string    `json:"id"`
	PIDs      []int     `json:"pids,omitempty"`
	Port      int       `json:"port,omitempty"`
	StartedAt time.Time `json:"started_at"`
}

func runDir() (string, error) {
	root, err := registry.RootDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(root, "run")
	return dir, os.MkdirAll(dir, 0o755)
}

func saveRun(r RunRecord) {
	dir, err := runDir()
	if err != nil {
		return
	}
	data, _ := json.MarshalIndent(r, "", "  ")
	_ = os.WriteFile(filepath.Join(dir, r.ID+".json"), append(data, '\n'), 0o644)
}

func loadRun(id string) RunRecord {
	dir, err := runDir()
	if err != nil {
		return RunRecord{ID: id}
	}
	data, err := os.ReadFile(filepath.Join(dir, id+".json"))
	if err != nil {
		return RunRecord{ID: id}
	}
	var r RunRecord
	if json.Unmarshal(data, &r) != nil {
		return RunRecord{ID: id}
	}
	return r
}

func clearRun(id string) {
	dir, err := runDir()
	if err != nil {
		return
	}
	_ = os.Remove(filepath.Join(dir, id+".json"))
}

func Launch(t registry.Tool) error {
	t = t.ForOS(runtime.GOOS)
	t.Workdir = platform.Expand(t.Workdir)
	t.URL = platform.Expand(t.URL)
	t.HealthURL = platform.Expand(t.HealthURL)
	t.AppPath = platform.Expand(t.AppPath)
	for i, c := range t.Command {
		t.Command[i] = platform.Expand(c)
	}
	if !t.Supports(runtime.GOOS) {
		err := fmt.Errorf("「%s」不支持当前系统 %s（支援：%s）", t.Name, registry.PlatformLabel(runtime.GOOS), labels(t.Platforms))
		applog.Warn("launch_skip", "id", t.ID, "err", err)
		return err
	}

	applog.Info("launch_begin", "id", t.ID, "kind", t.Kind, "workdir", t.Workdir, "url", t.URL, "app", t.AppPath)
	var err error
	switch t.Kind {
	case "url":
		err = platform.OpenURL(t.URL)
	case "app":
		err = platform.OpenPath(t.AppPath)
	case "command":
		err = launchCommand(t)
	case "web":
		err = launchWeb(t)
	default:
		err = fmt.Errorf("unknown kind %q", t.Kind)
	}
	if err != nil {
		applog.Error("launch_end", "id", t.ID, "err", err)
		return err
	}
	applog.Info("launch_ok", "id", t.ID)
	return nil
}

func launchCommand(t registry.Tool) error {
	if t.Terminal {
		if err := platform.RunInTerminal(t.Workdir, t.Command, t.Env); err != nil {
			return err
		}
		time.Sleep(800 * time.Millisecond)
		saveCollected(t, 0)
		return nil
	}
	pid, err := platform.StartDetached(t.Workdir, t.Command, t.Env)
	if err != nil {
		return err
	}
	saveCollected(t, pid)
	return nil
}

func launchWeb(t registry.Tool) error {
	st := Probe(t)
	if st.Running {
		return platform.OpenURL(t.URL)
	}
	port := proc.PortFromURL(t.URL)
	if occupied, who := portBusy(t.URL); occupied && !st.Running {
		listeners := proc.PIDsOnPort(port)
		if len(listeners) > 0 {
			return fmt.Errorf("端口已被占用（%s）。请先点「关闭」停掉占用进程，再打开「%s」", who, t.Name)
		}
	}
	var startedPID int
	if len(t.Command) > 0 {
		var err error
		if t.Terminal {
			err = platform.RunInTerminal(t.Workdir, t.Command, t.Env)
		} else {
			startedPID, err = platform.StartDetached(t.Workdir, t.Command, t.Env)
		}
		if err != nil {
			return err
		}
		_ = waitHealthy(t, 25*time.Second)
		saveCollected(t, startedPID)
		return platform.OpenURL(t.URL)
	}
	return platform.OpenURL(t.URL)
}

func saveCollected(t registry.Tool, extraPID int) {
	port := proc.PortFromURL(t.URL)
	if port == 0 {
		port = proc.PortFromURL(t.HealthURL)
	}
	pids := []int{}
	if extraPID > 0 {
		pids = append(pids, extraPID)
	}
	pids = append(pids, proc.PIDsOnPort(port)...)
	if n := needle(t); n != "" {
		pids = append(pids, proc.FindByNeedle(n)...)
	}
	saveRun(RunRecord{ID: t.ID, PIDs: uniqueInts(pids), Port: port, StartedAt: time.Now()})
}

func Stop(t registry.Tool) error {
	applog.Info("stop_begin", "id", t.ID, "kind", t.Kind, "name", t.Name)
	t = t.ForOS(runtime.GOOS)
	t.Workdir = platform.Expand(t.Workdir)
	t.URL = platform.Expand(t.URL)
	t.AppPath = platform.Expand(t.AppPath)

	if len(t.StopCommand) > 0 {
		_, _ = platform.StartDetached(t.Workdir, t.StopCommand, t.Env)
		time.Sleep(500 * time.Millisecond)
	}

	rec := loadRun(t.ID)
	var pids []int
	pids = append(pids, rec.PIDs...)

	port := rec.Port
	if port == 0 {
		port = proc.PortFromURL(t.URL)
	}
	if port == 17890 {
		port = 0 // never kill the hub
	}
	// Only kill whoever is on this port if it is actually THIS tool
	// (health match) or the tool has no health fingerprint.
	includePort := t.HealthContains == ""
	if !includePort && t.URL != "" {
		ok, _ := httpOK(t.URL, t.HealthContains)
		if t.HealthURL != "" {
			ok, _ = httpOK(t.HealthURL, t.HealthContains)
		}
		includePort = ok
	}
	if includePort && port > 0 {
		pids = append(pids, proc.PIDsOnPort(port)...)
	}
	if n := needle(t); n != "" {
		pids = append(pids, proc.FindByNeedle(n)...)
	}
	if t.Kind == "app" {
		proc.QuitApp(t.AppPath)
		if n := filepath.Base(strings.TrimSuffix(t.AppPath, filepath.Ext(t.AppPath))); n != "" {
			pids = append(pids, proc.FindByNeedle(n)...)
		}
	}

	killed := proc.KillAll(pids)
	if includePort && port > 0 {
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) && len(proc.PIDsOnPort(port)) > 0 {
			_ = proc.KillAll(proc.PIDsOnPort(port))
			time.Sleep(150 * time.Millisecond)
		}
	}
	clearRun(t.ID)
	applog.Info("stop_ok", "id", t.ID, "killed", killed, "port", port)
	return nil
}

func needle(t registry.Tool) string {
	if strings.TrimSpace(t.ProcessMatch) != "" {
		return t.ProcessMatch
	}
	return proc.DistinctiveNeedle(t.Command)
}

func uniqueInts(in []int) []int {
	seen := map[int]struct{}{}
	out := make([]int, 0, len(in))
	for _, n := range in {
		if n <= 0 {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	return out
}

func labels(platforms []string) string {
	parts := make([]string, 0, len(platforms))
	for _, p := range platforms {
		parts = append(parts, registry.PlatformLabel(p))
	}
	return strings.Join(parts, " / ")
}

func ownPIDs(t registry.Tool, includePort bool) []int {
	rec := loadRun(t.ID)
	var pids []int
	for _, pid := range rec.PIDs {
		if proc.Alive(pid) && pid != proc.SelfPID() {
			pids = append(pids, pid)
		}
	}
	if n := needle(t); n != "" {
		pids = append(pids, proc.FindByNeedle(n)...)
	}
	if includePort {
		port := rec.Port
		if port == 0 {
			port = proc.PortFromURL(t.URL)
		}
		if port > 0 && port != 17890 {
			pids = append(pids, proc.PIDsOnPort(port)...)
		}
	}
	live := make([]int, 0)
	for _, pid := range uniqueInts(pids) {
		if proc.Alive(pid) && pid != proc.SelfPID() {
			live = append(live, pid)
		}
	}
	return live
}

func CollectedPIDs(t registry.Tool) []int {
	u := t.HealthURL
	if u == "" {
		u = t.URL
	}
	includePort := u == ""
	if u != "" {
		ok, _ := httpOK(u, t.HealthContains)
		includePort = ok || t.HealthContains == ""
	}
	return ownPIDs(t, includePort)
}
