package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ts721521/DevToolbox/internal/registry"
)

func TestMoveToolsAPI(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("APPDATA", dir)
	t.Setenv("USERPROFILE", dir)

	if err := registry.Save(registry.Tool{ID: "x", Name: "X", Kind: "url", URL: "http://127.0.0.1:9", Group: "工作"}); err != nil {
		t.Fatal(err)
	}
	h := Handler(nil, Options{})
	body, _ := json.Marshal(map[string]any{"ids": []string{"x"}, "group": "财务"})
	req := hubRequest(http.MethodPost, "/api/tools/move", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", rr.Code, rr.Body.String())
	}
	got, err := registry.Get("x")
	if err != nil || got.Group != "财务" {
		t.Fatalf("got %+v err=%v", got, err)
	}
}

func TestTabsRenameAndDeleteAPI(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("APPDATA", dir)
	t.Setenv("USERPROFILE", dir)

	h := Handler(nil, Options{})

	add, _ := json.Marshal(map[string]string{"name": "临时"})
	req := hubRequest(http.MethodPost, "/api/tabs", bytes.NewReader(add))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("add status %d body=%s", rr.Code, rr.Body.String())
	}

	rename, _ := json.Marshal(map[string]string{"from": "临时", "to": "临时二"})
	req = hubRequest(http.MethodPost, "/api/tabs/rename", bytes.NewReader(rename))
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), "临时二") {
		t.Fatalf("rename status %d body=%s", rr.Code, rr.Body.String())
	}

	req = hubRequest(http.MethodDelete, "/api/tabs/"+url.PathEscape("临时二")+"?move="+url.QueryEscape("其他"), nil)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("delete status %d body=%s", rr.Code, rr.Body.String())
	}

	req = hubRequest(http.MethodDelete, "/api/tabs/"+url.PathEscape("不存在"), nil)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("missing tab status %d body=%s", rr.Code, rr.Body.String())
	}
}
