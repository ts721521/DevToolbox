package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ts721521/DevToolbox/internal/registry"
)

func TestBlockedRegisterAndRestoreHTTP(t *testing.T) {
	withTempHome(t)
	h := Handler(nil, Options{})
	body := `{"id":"blocked-one","name":"B","kind":"url","url":"http://127.0.0.1:9"}`
	req := hubRequest(http.MethodPost, "/api/tools", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("register %d %s", rr.Code, rr.Body.String())
	}

	req = hubRequest(http.MethodDelete, "/api/tools/blocked-one", nil)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("delete %d %s", rr.Code, rr.Body.String())
	}

	req = hubRequest(http.MethodPost, "/api/tools", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("blocked register want 409 got %d %s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/blocked", nil)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("list blocked %d", rr.Code)
	}
	var listed struct {
		Entries []registry.BlockedEntry `json:"entries"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Entries) != 1 || listed.Entries[0].ID != "blocked-one" {
		t.Fatalf("entries %+v", listed.Entries)
	}

	req = hubRequest(http.MethodPost, "/api/tools/blocked-one/restore", nil)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("restore %d %s", rr.Code, rr.Body.String())
	}
	if _, err := registry.Get("blocked-one"); err != nil {
		t.Fatal(err)
	}
}

func TestUnblockHTTP(t *testing.T) {
	withTempHome(t)
	h := Handler(nil, Options{})
	body := `{"id":"gone-http","name":"G","kind":"url","url":"http://127.0.0.1:9"}`
	req := hubRequest(http.MethodPost, "/api/tools", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatal(rr.Body.String())
	}
	req = hubRequest(http.MethodDelete, "/api/tools/gone-http", nil)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	req = hubRequest(http.MethodDelete, "/api/blocked/gone-http", nil)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("unblock %d %s", rr.Code, rr.Body.String())
	}
	req = hubRequest(http.MethodPost, "/api/tools", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("re-register after unblock %d %s", rr.Code, rr.Body.String())
	}
}

func TestBlockedWorkdirHTTP(t *testing.T) {
	withTempHome(t)
	h := Handler(nil, Options{})
	wd := strings.ReplaceAll(t.TempDir(), `\`, `/`)
	body := fmt.Sprintf(`{"id":"wd1","name":"W","kind":"url","url":"http://127.0.0.1:9","workdir":%q}`, wd)
	req := hubRequest(http.MethodPost, "/api/tools", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatal(rr.Body.String())
	}
	req = hubRequest(http.MethodDelete, "/api/tools/wd1", nil)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	body2 := fmt.Sprintf(`{"id":"wd2","name":"W2","kind":"url","url":"http://127.0.0.1:10","workdir":%q}`, wd)
	req = hubRequest(http.MethodPost, "/api/tools", strings.NewReader(body2))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("workdir block want 409 got %d %s", rr.Code, rr.Body.String())
	}
}

func TestUnblockTraversalHTTP(t *testing.T) {
	h := Handler(nil, Options{})
	req := hubRequest(http.MethodDelete, "/api/blocked/..%2fetc", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status %d body=%s", rr.Code, rr.Body.String())
	}
}
