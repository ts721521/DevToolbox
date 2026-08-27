package server

import (
	"net/http"

	"github.com/ts721521/DevToolbox/internal/applog"
	"github.com/ts721521/DevToolbox/internal/registry"
)

func handleBlockedList(w http.ResponseWriter, _ *http.Request) {
	ents, err := registry.ListBlocked()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	type item struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Workdir  string `json:"workdir,omitempty"`
		Git      string `json:"git,omitempty"`
		GitLabel string `json:"git_label,omitempty"`
		BuriedAt string `json:"buried_at,omitempty"`
	}
	out := make([]item, 0, len(ents))
	for _, e := range ents {
		it := item{ID: e.ID, Name: e.Name, Workdir: e.Workdir, Git: e.Git}
		if !e.BuriedAt.IsZero() {
			it.BuriedAt = e.BuriedAt.UTC().Format("2006-01-02T15:04:05Z")
		}
		if e.Git != "" {
			it.GitLabel = registry.GitLabel(e.Git)
		}
		out = append(out, it)
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": out})
}

func handleRestore(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := registry.Restore(id); err != nil {
		applog.Warn("restore", "id", id, "err", err)
		http.Error(w, err.Error(), httpStatus(err))
		return
	}
	applog.Info("restore", "id", id)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": id})
}

func handleUnblock(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := registry.Unblock(id); err != nil {
		applog.Warn("unblock", "id", id, "err", err)
		http.Error(w, err.Error(), httpStatus(err))
		return
	}
	applog.Info("unblock", "id", id)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": id})
}
