package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/ts721521/DevToolbox/internal/proc"
	"github.com/ts721521/DevToolbox/internal/registry"
)

const lockName = "hub.lock"

// HubLock records who currently owns :17890 so a new launch can
// either hand off to that process or replace a stale/old binary.
type HubLock struct {
	PID     int    `json:"pid"`
	Exe     string `json:"exe"`
	Mtime   int64  `json:"mtime"`
	Version string `json:"version"`
	Commit  string `json:"commit"`
}

// HubIdentity is what the process already listening on Addr reports over HTTP.
type HubIdentity struct {
	Name    string
	Version string
	Commit  string
	Exe     string
	Dock    bool
}

func HubPort() int {
	_, p, err := net.SplitHostPort(Addr)
	if err != nil {
		return 17890
	}
	n, err := strconv.Atoi(p)
	if err != nil || n <= 0 {
		return 17890
	}
	return n
}

func lockPath() (string, error) {
	root, err := registry.RootDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(root, lockName), nil
}

func ExeMtime(exe string) int64 {
	st, err := os.Stat(exe)
	if err != nil {
		return 0
	}
	return st.ModTime().UnixNano()
}

func lockUsable(lock HubLock) bool {
	return lock.PID > 0 && lock.PID != proc.SelfPID() && proc.Alive(lock.PID)
}

// shouldHandoff is true when the live hub is the same build (version+commit).
// In-place replace of the same exe path with a newer mtime is takeover.
func shouldHandoff(lock HubLock, lockOK bool, live HubIdentity, exe, ver, commit string) bool {
	if !live.Dock {
		return false
	}
	if live.Version != ver || live.Commit != commit {
		return false
	}
	if !lockOK || !lockUsable(lock) {
		return true
	}
	if lock.Version != ver || lock.Commit != commit {
		return false
	}
	mtime := ExeMtime(exe)
	if lock.Exe == exe && mtime != 0 && lock.Mtime != 0 && lock.Mtime != mtime {
		return false
	}
	return true
}

func containsPID(pids []int, pid int) bool {
	for _, p := range pids {
		if p == pid {
			return true
		}
	}
	return false
}

func pidsToStop(lock HubLock, lockOK bool, portPIDs []int) []int {
	if lockOK && lockUsable(lock) && containsPID(portPIDs, lock.PID) {
		return []int{lock.PID}
	}
	return portPIDs
}

func readLock() (HubLock, error) {
	p, err := lockPath()
	if err != nil {
		return HubLock{}, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return HubLock{}, err
	}
	var l HubLock
	if err := json.Unmarshal(data, &l); err != nil {
		return HubLock{}, err
	}
	return l, nil
}

func WriteLock(exe, ver, commit string) error {
	p, err := lockPath()
	if err != nil {
		return err
	}
	l := HubLock{
		PID:     proc.SelfPID(),
		Exe:     exe,
		Mtime:   ExeMtime(exe),
		Version: ver,
		Commit:  commit,
	}
	data, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, append(data, '\n'), 0o644)
}

func hubClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func getJSON(client *http.Client, path string, max int64) (map[string]any, error) {
	resp, err := client.Get("http://" + Addr + path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, max))
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		return nil, err
	}
	return got, nil
}

func PeekHub() (name, ver string, ok bool) {
	id := PeekIdentity()
	return id.Name, id.Version, id.Dock
}

func PeekIdentity() HubIdentity {
	if !AlreadyRunning() {
		return HubIdentity{}
	}
	client := hubClient(400 * time.Millisecond)
	if got, err := getJSON(client, "/api/system", 64<<10); err == nil {
		name, _ := got["name"].(string)
		ver, _ := got["version"].(string)
		commit, _ := got["commit"].(string)
		exe, _ := got["executable"].(string)
		if name == "ToolDock" {
			return HubIdentity{Name: name, Version: ver, Commit: commit, Exe: exe, Dock: true}
		}
		if name != "" {
			return HubIdentity{Name: name}
		}
	}
	got, err := getJSON(client, "/api/health", 8<<10)
	if err != nil {
		return HubIdentity{}
	}
	name, _ := got["name"].(string)
	ver, _ := got["version"].(string)
	commit, _ := got["commit"].(string)
	return HubIdentity{Name: name, Version: ver, Commit: commit, Dock: name == "ToolDock"}
}

func stopOccupant() {
	client := hubClient(800 * time.Millisecond)
	token := ""
	if got, err := getJSON(client, "/api/system", 64<<10); err == nil {
		if s, ok := got["token"].(string); ok {
			token = s
		}
	}
	req, err := http.NewRequest(http.MethodPost, "http://"+Addr+"/api/shutdown", nil)
	if err == nil {
		req.Header.Set("Origin", "http://"+Addr)
		if token != "" {
			req.Header.Set("X-ToolDock-Token", token)
		}
		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
		}
	}
	deadline := time.Now().Add(2 * time.Second)
	for AlreadyRunning() && time.Now().Before(deadline) {
		time.Sleep(80 * time.Millisecond)
	}
	if !AlreadyRunning() {
		return
	}
	live := PeekIdentity()
	if !live.Dock {
		return
	}
	portPIDs := proc.PIDsOnPort(HubPort())
	lock, lockErr := readLock()
	_ = proc.KillAll(pidsToStop(lock, lockErr == nil, portPIDs))
	deadline = time.Now().Add(2 * time.Second)
	for AlreadyRunning() && time.Now().Before(deadline) {
		time.Sleep(80 * time.Millisecond)
	}
}

var hubOccupied = AlreadyRunning
var peekLive = PeekIdentity
var stopHub = stopOccupant

// ClaimHub enforces a single ToolDock on :17890.
// handoff means a live copy of this build already owns the port:
// the caller should open the UI and exit. The caller must WriteLock
// only after Listen succeeds.
func ClaimHub(exe, ver, commit string) (handoff bool, err error) {
	if !hubOccupied() {
		return false, nil
	}
	live := peekLive()
	if !live.Dock {
		name := live.Name
		if name == "" {
			name = "unknown"
		}
		return false, fmt.Errorf("%s 已被其他程序占用（%s）", Addr, name)
	}
	lock, lockErr := readLock()
	if shouldHandoff(lock, lockErr == nil, live, exe, ver, commit) {
		return true, nil
	}
	stopHub()
	if hubOccupied() {
		again := peekLive()
		if again.Dock && shouldHandoff(lock, lockErr == nil, again, exe, ver, commit) {
			return true, nil
		}
		if !again.Dock && hubOccupied() {
			return false, fmt.Errorf("%s 已被其他程序占用", Addr)
		}
		return false, fmt.Errorf("无法接管 %s 上的旧工坞进程", Addr)
	}
	return false, nil
}

func handleShutdown(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	go afterShutdown()
}

var afterShutdown = func() {
	time.Sleep(120 * time.Millisecond)
	os.Exit(0)
}
