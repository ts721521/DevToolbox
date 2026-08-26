package server

import (
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/ts721521/DevToolbox/internal/applog"
	"github.com/ts721521/DevToolbox/internal/platform"
)

type statusWriter struct {
	http.ResponseWriter
	code  int
	wrote bool
}

func (w *statusWriter) WriteHeader(code int) {
	if w.wrote {
		return
	}
	w.code = code
	w.wrote = true
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(p []byte) (int, error) {
	if !w.wrote {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(p)
}

func withLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, code: http.StatusOK}
		panicked := false
		defer func() {
			if rec := recover(); rec != nil {
				panicked = true
				if !sw.wrote {
					http.Error(sw, "internal error", http.StatusInternalServerError)
				} else {
					sw.code = http.StatusInternalServerError
				}
				applog.Error("panic", "path", r.URL.Path, "err", rec, "stack", string(debug.Stack()))
			}
		}()
		next.ServeHTTP(sw, r)
		code := sw.code
		if panicked && code < 500 {
			code = http.StatusInternalServerError
		}
		if !shouldLogHTTP(r, code) {
			return
		}
		if code >= 500 {
			applog.Error("http", "method", r.Method, "path", r.URL.Path, "status", code, "ms", time.Since(start).Milliseconds())
			return
		}
		if code >= 400 {
			applog.Warn("http", "method", r.Method, "path", r.URL.Path, "status", code, "ms", time.Since(start).Milliseconds())
			return
		}
		applog.Info("http", "method", r.Method, "path", r.URL.Path, "status", code, "ms", time.Since(start).Milliseconds())
	})
}

func shouldLogHTTP(r *http.Request, code int) bool {
	path := r.URL.Path
	if path == "/json/version" || path == "/json/list" || path == "/favicon.ico" {
		return false
	}
	if code >= 400 {
		return true
	}
	if r.Method == http.MethodGet {
		switch path {
		case "/api/tools", "/api/health", "/api/logs", "/api/tabs", "/style.css", "/app.js", "/icon.png":
			return false
		}
		if strings.HasSuffix(path, ".css") || strings.HasSuffix(path, ".js") || strings.HasSuffix(path, ".png") || strings.HasSuffix(path, ".ico") {
			return false
		}
	}
	return true
}

func handleLogs(w http.ResponseWriter, r *http.Request) {
	n := 300
	text, err := applog.Tail(n)
	if err != nil {
		applog.Error("logs_read", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	p, _ := applog.Path()
	d, _ := applog.Dir()
	writeJSON(w, http.StatusOK, map[string]any{
		"path": p,
		"dir":  d,
		"text": text,
	})
}

func handleLogsOpen(w http.ResponseWriter, r *http.Request) {
	d, err := applog.Dir()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := platform.OpenPath(d); err != nil {
		applog.Error("logs_open", "err", err, "dir", d)
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	applog.Info("logs_open", "dir", d)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "dir": d})
}
