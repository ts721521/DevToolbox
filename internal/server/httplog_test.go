package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ts721521/DevToolbox/internal/applog"
)

func TestLogsEndpoint(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("APPDATA", dir)
	t.Setenv("USERPROFILE", dir)
	if err := applog.Init(); err != nil {
		t.Fatal(err)
	}
	applog.Info("probe_log", "ok", true)

	h := Handler(nil, Options{})
	req := httptest.NewRequest(http.MethodGet, "/api/logs", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "probe_log") {
		t.Fatalf("body=%s", rr.Body.String())
	}
}

func TestShouldSkipPolling(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/json/version", nil)
	if shouldLogHTTP(req, 404) {
		t.Fatal("should skip chrome inspector")
	}
	req = httptest.NewRequest(http.MethodGet, "/api/tools", nil)
	if shouldLogHTTP(req, 200) {
		t.Fatal("should skip GET /api/tools 200")
	}
	if !shouldLogHTTP(req, 500) {
		t.Fatal("should log errors")
	}
}

func TestPanicRecoverDoesNotDoubleWrite(t *testing.T) {
	h := withLog(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/boom", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status %d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "internal error") {
		t.Fatalf("body=%s", rr.Body.String())
	}
}

func TestPanicAfterWriteDoesNotDoubleBody(t *testing.T) {
	h := withLog(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("partial"))
		panic("later")
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/boom-late", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if !strings.HasPrefix(rr.Body.String(), "partial") {
		t.Fatalf("body=%s", rr.Body.String())
	}
	if strings.Count(rr.Body.String(), "internal error") != 0 {
		t.Fatalf("double write body=%s", rr.Body.String())
	}
}
