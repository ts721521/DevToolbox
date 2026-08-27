package launcher

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ts721521/DevToolbox/internal/applog"
	"github.com/ts721521/DevToolbox/internal/proc"
	"github.com/ts721521/DevToolbox/internal/registry"
)

func withTempHome(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("APPDATA", dir)
	t.Setenv("USERPROFILE", dir)
	t.Cleanup(applog.Close)
}

func allPlatforms() []string {
	return []string{"darwin", "linux", "windows"}
}

func processLive(rec RunRecord, needle string) bool {
	if needle != "" && len(proc.FindByNeedle(needle)) > 0 {
		return true
	}
	for _, pid := range append(append([]int{}, rec.PIDs...), rec.ServicePIDs...) {
		if proc.Alive(pid) {
			return true
		}
	}
	return false
}

func TestLaunchMissingDepend(t *testing.T) {
	withTempHome(t)
	tool := registry.Tool{
		ID:        "front",
		Name:      "Front",
		Kind:      "url",
		URL:       "http://127.0.0.1:9",
		Platforms: allPlatforms(),
		Depends:   []string{"redis"},
	}
	if err := registry.Save(tool); err != nil {
		t.Fatal(err)
	}
	if err := Launch(tool); err == nil {
		t.Fatal("expected missing depend")
	}
}

func TestLaunchDependCycle(t *testing.T) {
	withTempHome(t)
	a := registry.Tool{ID: "a", Name: "A", Kind: "url", URL: "http://127.0.0.1:9", Platforms: allPlatforms(), Depends: []string{"b"}}
	b := registry.Tool{ID: "b", Name: "B", Kind: "url", URL: "http://127.0.0.1:9", Platforms: allPlatforms(), Depends: []string{"a"}}
	if err := registry.Save(a); err != nil {
		t.Fatal(err)
	}
	if err := registry.Save(b); err != nil {
		t.Fatal(err)
	}
	if err := Launch(a); err == nil {
		t.Fatal("expected cycle")
	}
}

func TestLaunchDependAlreadyRunningStartsServices(t *testing.T) {
	withTempHome(t)
	wd := t.TempDir()
	script, needle := holdScript(t, wd)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	backend := registry.Tool{
		ID:        "backend",
		Name:      "Backend",
		Kind:      "url",
		URL:       srv.URL,
		Workdir:   wd,
		Platforms: allPlatforms(),
		Services: []registry.Service{{
			Name:         "hold",
			Command:      script,
			ProcessMatch: needle,
			WaitMS:       200,
		}},
	}
	front := registry.Tool{
		ID:        "front",
		Name:      "Front",
		Kind:      "url",
		URL:       "http://127.0.0.1:9",
		Platforms: allPlatforms(),
		Depends:   []string{"backend"},
	}
	if err := registry.Save(backend); err != nil {
		t.Fatal(err)
	}
	if err := registry.Save(front); err != nil {
		t.Fatal(err)
	}
	if err := launchChain(front, false, nil); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stopServices(prepare(backend)) })
	if !processLive(loadRun(backend.ID), needle) {
		t.Fatal("depend already running should still start its services")
	}
}

func TestStartServicesFailureStopsStarted(t *testing.T) {
	withTempHome(t)
	wd := t.TempDir()
	script, needle := holdScript(t, wd)
	tool := registry.Tool{
		ID:      "partial",
		Name:    "Partial",
		Kind:    "url",
		URL:     "http://127.0.0.1:9",
		Workdir: wd,
		Services: []registry.Service{
			{Name: "hold", Command: script, ProcessMatch: needle, WaitMS: 200},
			{Name: "bad", Command: []string{"__tooldock_no_such_command__"}},
		},
	}
	err := startServices(prepare(tool))
	if err == nil {
		t.Fatal("expected second service to fail")
	}
	t.Cleanup(func() { stopServices(prepare(tool)) })
	if n := proc.FindByNeedle(needle); len(n) > 0 {
		stopServices(prepare(tool))
		t.Fatalf("orphan service left running: %v", n)
	}
}

func TestStopDoesNotKillDepend(t *testing.T) {
	withTempHome(t)
	wd := t.TempDir()
	script, needle := holdScript(t, wd)
	dep := registry.Tool{
		ID:           "dep",
		Name:         "Dep",
		Kind:         "command",
		Workdir:      wd,
		Command:      script,
		ProcessMatch: needle,
		Platforms:    allPlatforms(),
	}
	front := registry.Tool{
		ID:        "front2",
		Name:      "Front2",
		Kind:      "url",
		URL:       "http://127.0.0.1:9",
		Platforms: allPlatforms(),
		Depends:   []string{"dep"},
	}
	if err := registry.Save(dep); err != nil {
		t.Fatal(err)
	}
	if err := registry.Save(front); err != nil {
		t.Fatal(err)
	}
	if err := launchChain(front, false, nil); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = Stop(dep) })
	if !processLive(loadRun(dep.ID), needle) {
		t.Fatal("depend should be running")
	}
	if err := Stop(front); err != nil {
		t.Fatal(err)
	}
	if n := proc.FindByNeedle(needle); len(n) == 0 {
		t.Fatal("stop front must not kill depend")
	}
}

func TestWaitServiceHealthTimeout(t *testing.T) {
	withTempHome(t)
	wd := t.TempDir()
	script, needle := holdScript(t, wd)
	tool := registry.Tool{
		ID:      "health-timeout",
		Name:    "HT",
		Kind:    "url",
		URL:     "http://127.0.0.1:9",
		Workdir: wd,
		Services: []registry.Service{{
			Name:         "hold",
			Command:      script,
			ProcessMatch: needle,
			HealthURL:    "http://127.0.0.1:1/",
			WaitMS:       400,
		}},
	}
	err := startServices(prepare(tool))
	if err == nil {
		t.Fatal("expected health timeout")
	}
	t.Cleanup(func() { stopServices(prepare(tool)) })
	if n := proc.FindByNeedle(needle); len(n) > 0 {
		stopServices(prepare(tool))
		t.Fatalf("orphan after health timeout: %v", n)
	}
	if rec := loadRun(tool.ID); len(rec.ServicePIDs) > 0 {
		t.Fatalf("stale service pids after rollback: %v", rec.ServicePIDs)
	}
}

func TestLaunchKindFailureClearsServicePIDs(t *testing.T) {
	withTempHome(t)
	wd := t.TempDir()
	script, needle := holdScript(t, wd)
	tool := registry.Tool{
		ID:        "fail-kind",
		Name:      "F",
		Kind:      "command",
		Workdir:   wd,
		Platforms: allPlatforms(),
		Command:   []string{"/no/such/tooldock-bin-xyz"},
		Services: []registry.Service{{
			Name:         "hold",
			Command:      script,
			ProcessMatch: needle,
			WaitMS:       200,
		}},
	}
	if err := registry.Save(tool); err != nil {
		t.Fatal(err)
	}
	if err := Launch(tool); err == nil {
		t.Fatal("expected launchKind fail")
	}
	t.Cleanup(func() { stopServices(prepare(tool)) })
	if n := proc.FindByNeedle(needle); len(n) > 0 {
		t.Fatalf("orphan after launchKind fail: %v", n)
	}
	if rec := loadRun(tool.ID); len(rec.ServicePIDs) > 0 {
		t.Fatalf("stale service pids: %v", rec.ServicePIDs)
	}
}

func TestWaitServiceHealthOK(t *testing.T) {
	withTempHome(t)
	wd := t.TempDir()
	script, needle := holdScript(t, wd)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	}))
	t.Cleanup(srv.Close)
	tool := registry.Tool{
		ID:      "health-ok",
		Name:    "HO",
		Kind:    "url",
		URL:     "http://127.0.0.1:9",
		Workdir: wd,
		Services: []registry.Service{{
			Name:           "hold",
			Command:        script,
			ProcessMatch:   needle,
			HealthURL:      srv.URL,
			HealthContains: "ready",
			WaitMS:         5000,
		}},
	}
	if err := startServices(prepare(tool)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stopServices(prepare(tool)) })
}

func TestStartAndStopServices(t *testing.T) {
	withTempHome(t)
	wd := t.TempDir()
	script, needle := holdScript(t, wd)
	tool := registry.Tool{
		ID:      "with-svc",
		Name:    "With",
		Kind:    "url",
		URL:     "http://127.0.0.1:9",
		Workdir: wd,
		Services: []registry.Service{{
			Name:         "hold",
			Command:      script,
			ProcessMatch: needle,
			WaitMS:       200,
		}},
	}
	if err := startServices(prepare(tool)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stopServices(prepare(tool)) })
	rec := loadRun(tool.ID)
	if len(rec.ServicePIDs) == 0 && len(proc.FindByNeedle(needle)) == 0 {
		t.Fatal("service process not found")
	}
	stopServices(prepare(tool))
	if n := proc.FindByNeedle(needle); len(n) > 0 {
		t.Fatalf("still running: %v", n)
	}
}

func TestProbeSeesLiveServicePIDs(t *testing.T) {
	withTempHome(t)
	ppid := os.Getppid()
	if ppid <= 0 || ppid == proc.SelfPID() || !proc.Alive(ppid) {
		t.Skip("need another live pid")
	}
	tool := registry.Tool{
		ID:           "svcprobe",
		Name:         "Svc",
		Kind:         "command",
		Command:      []string{"true"},
		ProcessMatch: "zzzz-no-match-9f3a2c",
	}
	saveRun(RunRecord{ID: tool.ID, ServicePIDs: []int{ppid}})
	st := Alive(tool)
	if !st.Running {
		t.Fatalf("services-only should count as running: %+v", st)
	}
	if Probe(tool).Running {
		t.Fatal("Probe must stay main-process-only so launchKind still starts")
	}
}

func holdScript(t *testing.T, dir string) (command []string, needle string) {
	t.Helper()
	needle = "hold-" + strings.ReplaceAll(t.Name(), "/", "-")
	if runtime.GOOS == "windows" {
		line := "ping 127.0.0.1 -n 40>nul & rem " + needle
		return []string{"cmd", "/c", line}, needle
	}
	name := needle + ".sh"
	p := filepath.Join(dir, name)
	body := "#!/bin/sh\nsleep 40\n"
	if err := os.WriteFile(p, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return []string{p}, needle
}
