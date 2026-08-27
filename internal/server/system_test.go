package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ts721521/DevToolbox/internal/version"
)

func TestHealthIncludesVersion(t *testing.T) {
	h := Handler(nil, Options{})
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d", rr.Code)
	}
	var got map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["ok"] != true || got["version"] != version.Display() {
		t.Fatalf("got %+v", got)
	}
	if got["commit"] != version.Commit {
		t.Fatalf("commit=%v", got["commit"])
	}
	if _, ok := got["token"]; ok {
		t.Fatal("health must not include csrf token")
	}
}

func TestSystemIncludesVersion(t *testing.T) {
	withTempHome(t)
	h := Handler(nil, Options{})
	req := httptest.NewRequest(http.MethodGet, "/api/system", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", rr.Code, rr.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["version"] != version.Display() {
		t.Fatalf("version=%v want %s", got["version"], version.Display())
	}
	if got["version_numeric"] != version.Numeric() {
		t.Fatalf("numeric=%v", got["version_numeric"])
	}
	if got["token"] != CSRFToken || CSRFToken == "" {
		t.Fatalf("token=%v", got["token"])
	}
}
