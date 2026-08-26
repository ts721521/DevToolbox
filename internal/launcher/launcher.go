package launcher

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ts721521/DevToolbox/internal/proc"
	"github.com/ts721521/DevToolbox/internal/registry"
)

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
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return false, err.Error()
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, "offline"
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if contains != "" && !strings.Contains(string(body), contains) {
		return false, "other service on this port"
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 500 {
		return true, "ok"
	}
	return false, resp.Status
}

func portBusy(raw string) (bool, string) {
	u, err := url.Parse(raw)
	if err != nil {
		return false, ""
	}
	host := u.Host
	if !strings.Contains(host, ":") {
		if u.Scheme == "https" {
			host += ":443"
		} else {
			host += ":80"
		}
	}
	c, err := net.DialTimeout("tcp", host, 300*time.Millisecond)
	if err != nil {
		return false, ""
	}
	_ = c.Close()
	_ = proc.PortFromURL(raw)
	return true, host
}
