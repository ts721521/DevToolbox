package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ts721521/DevToolbox/internal/applog"
	"github.com/ts721521/DevToolbox/internal/registry"
)

func withTempHome(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("APPDATA", dir)
	t.Setenv("USERPROFILE", dir)
	t.Cleanup(applog.Close)
}

func TestRevealDirMissingTool(t *testing.T) {
	withTempHome(t)
	h := Handler(nil, Options{})
	req := hubRequest(http.MethodPost, "/api/tools/no-such/dir", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestRevealDirWithoutWorkdir(t *testing.T) {
	withTempHome(t)
	if err := registry.Save(registry.Tool{ID: "bookmark", Name: "书签", Kind: "url", URL: "http://127.0.0.1:9"}); err != nil {
		t.Fatal(err)
	}
	h := Handler(nil, Options{})
	req := hubRequest(http.MethodPost, "/api/tools/bookmark/dir", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("status %d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "workdir") {
		t.Fatalf("body=%s", rr.Body.String())
	}
}

func TestRevealAppWithoutPath(t *testing.T) {
	withTempHome(t)
	if err := registry.Save(registry.Tool{ID: "bookmark", Name: "书签", Kind: "url", URL: "http://127.0.0.1:9"}); err != nil {
		t.Fatal(err)
	}
	h := Handler(nil, Options{})
	req := hubRequest(http.MethodPost, "/api/tools/bookmark/app", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("status %d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "app_path") {
		t.Fatalf("body=%s", rr.Body.String())
	}
}

func TestLaunchMissingDependAPI(t *testing.T) {
	withTempHome(t)
	if err := registry.Save(registry.Tool{
		ID: "front", Name: "Front", Kind: "url", URL: "http://127.0.0.1:9",
		Platforms: []string{"darwin", "linux", "windows"}, Depends: []string{"redis"},
	}); err != nil {
		t.Fatal(err)
	}
	h := Handler(nil, Options{})
	req := hubRequest(http.MethodPost, "/api/tools/front/launch", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("status %d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "redis") {
		t.Fatalf("body=%s", rr.Body.String())
	}
}

func TestRevealGitMissing(t *testing.T) {
	withTempHome(t)
	if err := registry.Save(registry.Tool{ID: "bookmark", Name: "书签", Kind: "url", URL: "http://127.0.0.1:9"}); err != nil {
		t.Fatal(err)
	}
	h := Handler(nil, Options{})
	req := hubRequest(http.MethodPost, "/api/tools/bookmark/git", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("status %d body=%s", rr.Code, rr.Body.String())
	}
}
