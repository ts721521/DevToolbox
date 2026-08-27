package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ts721521/DevToolbox/internal/proc"
)

func TestHubPort(t *testing.T) {
	if HubPort() != 17890 {
		t.Fatalf("port=%d", HubPort())
	}
}

func stubHub(t *testing.T, occupied bool, live HubIdentity, stop func()) {
	t.Helper()
	prevO, prevP, prevS := hubOccupied, peekLive, stopHub
	hubOccupied = func() bool { return occupied }
	peekLive = func() HubIdentity { return live }
	if stop != nil {
		stopHub = stop
	} else {
		stopHub = func() {}
	}
	t.Cleanup(func() {
		hubOccupied = prevO
		peekLive = prevP
		stopHub = prevS
	})
}

func TestShouldHandoffSameBuildDifferentPath(t *testing.T) {
	live := HubIdentity{Dock: true, Version: "v1.2.0", Commit: "abc"}
	lock := HubLock{PID: 1, Exe: "/Applications/工坞.app/Contents/MacOS/tooldock", Mtime: 9, Version: "v1.2.0", Commit: "abc"}
	if !shouldHandoff(lock, true, live, "/Users/x/bin/tooldock", "v1.2.0", "abc") {
		t.Fatal("cli vs app same build should handoff")
	}
}

func TestShouldHandoffMissingLockSameHTTP(t *testing.T) {
	live := HubIdentity{Dock: true, Version: "v1.2.0", Commit: "abc"}
	if !shouldHandoff(HubLock{}, false, live, "/bin/tooldock", "v1.2.0", "abc") {
		t.Fatal("missing lock + same advertised build must not kill")
	}
}

func TestShouldHandoffOlderVersion(t *testing.T) {
	live := HubIdentity{Dock: true, Version: "v1.1.0", Commit: "old"}
	if shouldHandoff(HubLock{}, false, live, "/bin/tooldock", "v1.2.0", "new") {
		t.Fatal("older hub should be replaced")
	}
}

func TestShouldHandoffInPlaceNewerBinary(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "tooldock")
	if err := os.WriteFile(exe, []byte("a"), 0o755); err != nil {
		t.Fatal(err)
	}
	old := ExeMtime(exe)
	if old == 0 {
		t.Fatal("mtime")
	}
	time.Sleep(20 * time.Millisecond)
	if err := os.WriteFile(exe, []byte("b"), 0o755); err != nil {
		t.Fatal(err)
	}
	ppid := os.Getppid()
	if ppid <= 0 || ppid == proc.SelfPID() || !proc.Alive(ppid) {
		t.Skip("need another live pid to model the occupant")
	}
	live := HubIdentity{Dock: true, Version: "v1.2.0-dev", Commit: "abc"}
	lock := HubLock{PID: ppid, Exe: exe, Mtime: old, Version: "v1.2.0-dev", Commit: "abc"}
	if shouldHandoff(lock, true, live, exe, "v1.2.0-dev", "abc") {
		t.Fatal("replaced binary at same path must take over")
	}
}

func TestPidsToStopPrefersLockOnlyIfOnPort(t *testing.T) {
	ppid := os.Getppid()
	if ppid <= 0 || ppid == proc.SelfPID() || !proc.Alive(ppid) {
		t.Skip("need another live pid")
	}
	lock := HubLock{PID: ppid}
	got := pidsToStop(lock, true, []int{7, ppid, 9})
	if len(got) != 1 || got[0] != ppid {
		t.Fatalf("got %v", got)
	}
	got = pidsToStop(lock, true, []int{7, 9})
	if len(got) != 2 || got[0] != 7 || got[1] != 9 {
		t.Fatalf("stale lock should not be killed: %v", got)
	}
	got = pidsToStop(HubLock{}, false, []int{7})
	if len(got) != 1 || got[0] != 7 {
		t.Fatalf("no lock: %v", got)
	}
}

func TestWriteLockRoundTrip(t *testing.T) {
	withTempHome(t)
	exe := filepath.Join(t.TempDir(), "tooldock")
	if err := os.WriteFile(exe, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := WriteLock(exe, "v1.2.0", "deadbeef"); err != nil {
		t.Fatal(err)
	}
	lock, err := readLock()
	if err != nil {
		t.Fatal(err)
	}
	if lock.PID != proc.SelfPID() || lock.Exe != exe || lock.Version != "v1.2.0" || lock.Commit != "deadbeef" {
		t.Fatalf("%+v", lock)
	}
	if lock.Mtime != ExeMtime(exe) {
		t.Fatalf("mtime %d vs %d", lock.Mtime, ExeMtime(exe))
	}
}

func TestClaimHubWhenPortFree(t *testing.T) {
	stubHub(t, false, HubIdentity{}, nil)
	handoff, err := ClaimHub("/bin/tooldock", "v1.2.0", "c")
	if err != nil || handoff {
		t.Fatalf("handoff=%v err=%v", handoff, err)
	}
}

func TestClaimHubHandoffSameBuild(t *testing.T) {
	withTempHome(t)
	stubHub(t, true, HubIdentity{Dock: true, Version: "v1.2.0", Commit: "abc"}, func() {
		t.Fatal("must not stop same build")
	})
	handoff, err := ClaimHub("/Users/x/bin/tooldock", "v1.2.0", "abc")
	if err != nil || !handoff {
		t.Fatalf("handoff=%v err=%v", handoff, err)
	}
}

func TestClaimHubRejectsForeignOccupant(t *testing.T) {
	stubHub(t, true, HubIdentity{Name: "nginx"}, func() {
		t.Fatal("must not kill foreign occupant")
	})
	_, err := ClaimHub("/bin/tooldock", "v1.2.0", "abc")
	if err == nil {
		t.Fatal("expected occupied error")
	}
}

func TestClaimHubTakeoverOlderBuild(t *testing.T) {
	withTempHome(t)
	stopped := false
	calls := 0
	prevO, prevP, prevS := hubOccupied, peekLive, stopHub
	t.Cleanup(func() {
		hubOccupied = prevO
		peekLive = prevP
		stopHub = prevS
	})
	hubOccupied = func() bool {
		calls++
		return !stopped
	}
	peekLive = func() HubIdentity {
		return HubIdentity{Dock: true, Version: "v1.1.0", Commit: "old"}
	}
	stopHub = func() { stopped = true }
	handoff, err := ClaimHub("/bin/tooldock", "v1.2.0", "new")
	if err != nil || handoff {
		t.Fatalf("handoff=%v err=%v", handoff, err)
	}
	if !stopped {
		t.Fatal("expected takeover")
	}
}

func TestShutdownRejectedWithoutOrigin(t *testing.T) {
	h := Handler(nil, Options{})
	req := httptest.NewRequest(http.MethodPost, "/api/shutdown", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status %d", rr.Code)
	}
}

func TestShutdownAllowsTokenWithoutOrigin(t *testing.T) {
	prev := afterShutdown
	afterShutdown = func() {}
	t.Cleanup(func() { afterShutdown = prev })
	h := Handler(nil, Options{})
	req := httptest.NewRequest(http.MethodPost, "/api/shutdown", nil)
	req.Header.Set("X-ToolDock-Token", CSRFToken)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestShutdownOKDoesNotExitInTest(t *testing.T) {
	prev := afterShutdown
	done := make(chan struct{})
	afterShutdown = func() { close(done) }
	t.Cleanup(func() { afterShutdown = prev })
	h := Handler(nil, Options{})
	req := hubRequest(http.MethodPost, "/api/shutdown", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", rr.Code, rr.Body.String())
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("shutdown hook")
	}
}
