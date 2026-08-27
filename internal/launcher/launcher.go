package launcher

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ts721521/DevToolbox/internal/proc"
	"github.com/ts721521/DevToolbox/internal/registry"
)

var healthHTTP = &http.Client{
	Timeout:       2 * time.Second,
	CheckRedirect: checkHealthRedirect,
}

func checkHealthRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 5 {
		return fmt.Errorf("too many redirects")
	}
	if req.URL == nil {
		return fmt.Errorf("redirect missing url")
	}
	if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
		return fmt.Errorf("redirect not http")
	}
	if !registry.LoopbackHost(req.URL.Hostname()) {
		return fmt.Errorf("redirect not local")
	}
	return nil
}

type Status struct {
	ID      string `json:"id"`
	Running bool   `json:"running"`
	Detail  string `json:"detail,omitempty"`
	PIDs    []int  `json:"pids,omitempty"`
}

func Probe(t registry.Tool) Status {
	u := t.HealthURL
	if u == "" {
		u = t.URL
	}
	if u != "" {
		ok, detail := httpOK(u, t.HealthContains)
		if ok {
			return Status{ID: t.ID, Running: true, Detail: detail, PIDs: ownPIDs(t, true)}
		}
		pids := ownPIDs(t, false)
		if len(pids) > 0 {
			return Status{ID: t.ID, Running: true, Detail: "process", PIDs: pids}
		}
		return Status{ID: t.ID, Running: false, Detail: detail}
	}
	pids := ownPIDs(t, false)
	if len(pids) > 0 {
		return Status{ID: t.ID, Running: true, Detail: "process", PIDs: pids}
	}
	return Status{ID: t.ID, Running: false, Detail: "stopped"}
}

// Alive is the dashboard/CLI "running" bit: main process or live services.
// Probe stays main-process-only so launchKind can still start the app itself.
func Alive(t registry.Tool) Status {
	st := Probe(t)
	if st.Running {
		return st
	}
	var pids []int
	for _, pid := range loadRun(t.ID).ServicePIDs {
		if proc.Alive(pid) && pid != proc.SelfPID() {
			pids = append(pids, pid)
		}
	}
	if len(pids) == 0 {
		return st
	}
	return Status{ID: t.ID, Running: true, Detail: "services", PIDs: pids}
}

func waitHealthy(t registry.Tool, d time.Duration) bool {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if Probe(t).Running {
			return true
		}
		time.Sleep(400 * time.Millisecond)
	}
	return false
}

func httpOK(raw, contains string) (bool, string) {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return false, "bad url"
	}
	if !registry.LoopbackHost(u.Hostname()) {
		return false, "not local"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return false, err.Error()
	}
	resp, err := healthHTTP.Do(req)
	if err != nil {
		return false, "offline"
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if contains != "" && !strings.Contains(string(body), contains) {
		return false, "other service on this port"
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return true, "ok"
	}
	return false, resp.Status
}

func portBusy(raw string) (bool, string) {
	port := proc.LocalListenPort(raw)
	if port == 0 {
		return false, ""
	}
	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	c, err := net.DialTimeout("tcp", addr, 300*time.Millisecond)
	if err != nil {
		return false, ""
	}
	_ = c.Close()
	return true, addr
}
