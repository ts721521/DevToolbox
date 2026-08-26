package server

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ts721521/DevToolbox/internal/desktop"
	"github.com/ts721521/DevToolbox/internal/launcher"
	"github.com/ts721521/DevToolbox/internal/registry"
	"github.com/ts721521/DevToolbox/internal/version"
)

const Addr = "127.0.0.1:17890"

type Options struct {
	Guides map[string][]byte
}

func Handler(static http.Handler, opt Options) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, 200, map[string]any{"ok": true, "name": "DevToolbox"})
	})
	mux.HandleFunc("GET /api/system", handleSystem)
	mux.HandleFunc("GET /api/tools", handleList)
	mux.HandleFunc("POST /api/tools", handleRegister)
	mux.HandleFunc("GET /api/tools/{id}", handleGet)
	mux.HandleFunc("DELETE /api/tools/{id}", handleDelete)
	mux.HandleFunc("POST /api/tools/{id}/launch", handleLaunch)
	mux.HandleFunc("POST /api/tools/{id}/stop", handleStop)
	for name, data := range opt.Guides {
		n, d := name, data
		mux.HandleFunc("GET /"+n, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
			_, _ = w.Write(d)
		})
	}
	if static != nil {
		mux.Handle("/", static)
	}
	return mux
}

func AlreadyRunning() bool {
	c, err := net.Dial("tcp", Addr)
	if err != nil {
		return false
	}
	_ = c.Close()
	return true
}

func handleSystem(w http.ResponseWriter, _ *http.Request) {
	root, _ := registry.RootDir()
	tools, _ := registry.Dir()
	self, _ := desktop.SelfPath()
	cli := ""
	if home, err := os.UserHomeDir(); err == nil {
		cli = filepath.Join(home, "bin", "devtoolbox")
		if runtime.GOOS == "windows" {
			cli += ".exe"
		}
	}
	writeJSON(w, 200, map[string]any{
		"name":            "DevToolbox",
		"version":         version.Version,
		"os":              runtime.GOOS,
		"os_name":         registry.PlatformLabel(runtime.GOOS),
		"arch":            runtime.GOARCH,
		"hub_platforms":   []string{"macOS", "Windows"},
		"hub_goos":        []string{"darwin", "windows"},
		"config_dir":      root,
		"tools_dir":       tools,
		"executable":      self,
		"cli":             cli,
		"ui":              URL(),
		"register_docs":   []string{"/AGENTS.md", "/HOW_TO_REGISTER.md"},
		"agents_md":       filepath.Join(root, "AGENTS.md"),
		"how_to_register": "Write .devtoolbox.json then: devtoolbox register --file .devtoolbox.json",
	})
}

func handleList(w http.ResponseWriter, _ *http.Request) {
	tools, err := registry.List()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	type item struct {
		registry.Tool
		Running    bool     `json:"running"`
		Detail     string   `json:"detail,omitempty"`
		PIDs       []int    `json:"pids,omitempty"`
		Compatible bool     `json:"compatible"`
		PlatformUI []string `json:"platform_labels"`
		CurrentOS  string   `json:"current_os"`
	}
	out := make([]item, 0, len(tools))
	for _, t := range tools {
		st := launcher.Probe(t)
		labels := make([]string, 0, len(t.Platforms))
		for _, p := range t.Platforms {
			labels = append(labels, registry.PlatformLabel(p))
		}
		out = append(out, item{
			Tool:       t,
			Running:    st.Running,
			Detail:     st.Detail,
			PIDs:       st.PIDs,
			Compatible: t.Supports(runtime.GOOS),
			PlatformUI: labels,
			CurrentOS:  registry.PlatformLabel(runtime.GOOS),
		})
	}
	writeJSON(w, 200, out)
}

func handleGet(w http.ResponseWriter, r *http.Request) {
	t, err := registry.Get(r.PathValue("id"))
	if err != nil {
		status := 400
		if errors.Is(err, registry.ErrNotFound) {
			status = 404
		}
		http.Error(w, err.Error(), status)
		return
	}
	writeJSON(w, 200, t)
}

func handleRegister(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	var one registry.Tool
	if err := json.Unmarshal(body, &one); err != nil {
		http.Error(w, "invalid json", 400)
		return
	}
	if err := registry.Save(one); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true, "id": one.ID})
}

func handleDelete(w http.ResponseWriter, r *http.Request) {
	if err := registry.Remove(r.PathValue("id")); err != nil {
		http.Error(w, err.Error(), 404)
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

func handleLaunch(w http.ResponseWriter, r *http.Request) {
	t, err := registry.Get(r.PathValue("id"))
	if err != nil {
		http.Error(w, err.Error(), 404)
		return
	}
	if err := launcher.Launch(t); err != nil {
		http.Error(w, err.Error(), 409)
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true, "id": t.ID})
}

func handleStop(w http.ResponseWriter, r *http.Request) {
	t, err := registry.Get(r.PathValue("id"))
	if err != nil {
		http.Error(w, err.Error(), 404)
		return
	}
	if err := launcher.Stop(t); err != nil {
		http.Error(w, err.Error(), 409)
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true, "id": t.ID, "running": launcher.Probe(t).Running})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func URL() string {
	return "http://" + Addr
}

func IsLocal(r *http.Request) bool {
	host := r.RemoteAddr
	return strings.HasPrefix(host, "127.0.0.1") || strings.HasPrefix(host, "[::1]")
}
