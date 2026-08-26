package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/ts721521/DevToolbox/internal/applog"
	"github.com/ts721521/DevToolbox/internal/registry"
)

func handleTabsGet(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"tabs":    registry.LoadTabs(),
		"builtin": registry.DefaultTabs,
		"other":   registry.DefaultOther,
	})
}

func handleTabsPut(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Tabs []string `json:"tabs"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if err := registry.ReplaceTabs(body.Tabs); err != nil {
		applog.Error("tabs_replace", "err", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	applog.Info("tabs_replace", "tabs", strings.Join(registry.LoadTabs(), ","))
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "tabs": registry.LoadTabs()})
}

func handleTabsAdd(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	name, err := registry.AddTab(body.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	applog.Info("tab_add", "name", name)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "name": name, "tabs": registry.LoadTabs()})
}

func handleTabsRename(w http.ResponseWriter, r *http.Request) {
	var body struct {
		From string `json:"from"`
		To   string `json:"to"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if err := registry.RenameTab(body.From, body.To); err != nil {
		applog.Warn("tab_rename", "from", body.From, "to", body.To, "err", err)
		writeTabError(w, err)
		return
	}
	applog.Info("tab_rename", "from", body.From, "to", body.To)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "to": registry.TabName(body.To), "tabs": registry.LoadTabs()})
}

func handleTabsDelete(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	move := r.URL.Query().Get("move")
	if err := registry.RemoveTab(name, move); err != nil {
		applog.Warn("tab_delete", "name", name, "err", err)
		writeTabError(w, err)
		return
	}
	applog.Info("tab_delete", "name", name, "move", move)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "tabs": registry.LoadTabs()})
}

func handleToolsMove(w http.ResponseWriter, r *http.Request) {
	var body struct {
		IDs   []string `json:"ids"`
		Group string   `json:"group"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if err := registry.MoveTools(body.IDs, body.Group); err != nil {
		applog.Warn("tools_move", "group", body.Group, "err", err)
		writeTabError(w, err)
		return
	}
	applog.Info("tools_move", "n", len(body.IDs), "group", body.Group)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "group": body.Group, "ids": body.IDs})
}

func writeTabError(w http.ResponseWriter, err error) {
	code := http.StatusBadRequest
	if errors.Is(err, registry.ErrNotFound) {
		code = http.StatusNotFound
	}
	http.Error(w, err.Error(), code)
}
