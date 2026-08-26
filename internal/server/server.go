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

	"github.com/ts721521/DevToolbox/internal/applog"
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
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "name": "ToolDock", "display": "工坞"})
	})
	mux.HandleFunc("GET /api/system", handleSystem)
	mux.HandleFunc("GET /api/ai-handoff", handleAIHandoff)
	mux.HandleFunc("GET /for-ai.md", handleForAI)
	mux.HandleFunc("GET /ai", handleForAI)
	mux.HandleFunc("GET /api/tabs", handleTabsGet)
	mux.HandleFunc("PUT /api/tabs", handleTabsPut)
	mux.HandleFunc("POST /api/tabs", handleTabsAdd)
	mux.HandleFunc("POST /api/tabs/rename", handleTabsRename)
	mux.HandleFunc("DELETE /api/tabs/{name}", handleTabsDelete)
	mux.HandleFunc("GET /api/tools", handleList)
	mux.HandleFunc("POST /api/tools", handleRegister)
	mux.HandleFunc("POST /api/tools/move", handleToolsMove)
	mux.HandleFunc("GET /api/tools/{id}", handleGet)
	mux.HandleFunc("DELETE /api/tools/{id}", handleDelete)
	mux.HandleFunc("POST /api/tools/{id}/launch", handleLaunch)
	mux.HandleFunc("POST /api/tools/{id}/stop", handleStop)
	mux.HandleFunc("POST /api/tools/{id}/dir", handleRevealDir)
	mux.HandleFunc("POST /api/tools/{id}/app", handleRevealApp)
	mux.HandleFunc("GET /api/logs", handleLogs)
	mux.HandleFunc("POST /api/logs/open", handleLogsOpen)
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
	return withLog(mux)
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
		cli = filepath.Join(home, "bin", "tooldock")
		if runtime.GOOS == "windows" {
			cli += ".exe"
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"name":            "ToolDock",
		"display_name":    "工坞",
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
		"register_docs":   []string{"/for-ai.md", "/AGENTS.md"},
		"ai_url":          ForAIURL(),
		"tabs":            registry.LoadTabs(),
		"builtin_tabs":    registry.DefaultTabs,
		"log_path":        logPath(),
		"log_dir":         logDir(),
		"agents_md":       filepath.Join(root, "AGENTS.md"),
		"how_to_register": "Write .devtoolbox.json then: tooldock register --file .devtoolbox.json",
	})
}

func handleList(w http.ResponseWriter, _ *http.Request) {
	tools, err := registry.List()
	if err != nil {
		applog.Error("list_tools", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
	writeJSON(w, http.StatusOK, out)
}

func handleGet(w http.ResponseWriter, r *http.Request) {
	t, err := registry.Get(r.PathValue("id"))
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, registry.ErrNotFound) {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func handleRegister(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var one registry.Tool
	if err := json.Unmarshal(body, &one); err != nil {
		applog.Warn("register_bad_json", "err", err)
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if err := registry.Save(one); err != nil {
		applog.Error("register", "id", one.ID, "err", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	applog.Info("register", "id", one.ID, "name", one.Name, "kind", one.Kind, "workdir", one.Workdir)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": one.ID})
}

func handleDelete(w http.ResponseWriter, r *http.Request) {
	if err := registry.Remove(r.PathValue("id")); err != nil {
		applog.Warn("unregister", "id", r.PathValue("id"), "err", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	applog.Info("unregister", "id", r.PathValue("id"))
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func handleLaunch(w http.ResponseWriter, r *http.Request) {
	t, err := registry.Get(r.PathValue("id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err := launcher.Launch(t); err != nil {
		applog.Error("launch", "id", t.ID, "name", t.Name, "kind", t.Kind, "err", err)
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	applog.Info("launch", "id", t.ID, "name", t.Name, "kind", t.Kind, "url", t.URL)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": t.ID})
}

func handleStop(w http.ResponseWriter, r *http.Request) {
	t, err := registry.Get(r.PathValue("id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err := launcher.Stop(t); err != nil {
		applog.Error("stop", "id", t.ID, "name", t.Name, "err", err)
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	applog.Info("stop", "id", t.ID, "name", t.Name)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": t.ID, "running": launcher.Probe(t).Running})
}

func handleRevealDir(w http.ResponseWriter, r *http.Request) {
	t, err := registry.Get(r.PathValue("id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err := launcher.RevealDir(t); err != nil {
		applog.Warn("reveal_dir", "id", t.ID, "err", err)
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": t.ID, "kind": "dir"})
}

func handleRevealApp(w http.ResponseWriter, r *http.Request) {
	t, err := registry.Get(r.PathValue("id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err := launcher.RevealProgram(t); err != nil {
		applog.Warn("reveal_app", "id", t.ID, "err", err)
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": t.ID, "kind": "app"})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func URL() string {
	return "http://" + Addr
}

func logPath() string {
	p, _ := applog.Path()
	return p
}

func logDir() string {
	d, _ := applog.Dir()
	return d
}

func IsLocal(r *http.Request) bool {
	host := r.RemoteAddr
	return strings.HasPrefix(host, "127.0.0.1") || strings.HasPrefix(host, "[::1]")
}
