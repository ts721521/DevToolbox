package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOriginRejectsForeignPOST(t *testing.T) {
	h := Handler(nil, Options{})
	req := httptest.NewRequest(http.MethodPost, "/api/tools/x/stop", nil)
	req.Header.Set("Origin", "https://evil.example")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestEmptyOriginNeedsToken(t *testing.T) {
	h := Handler(nil, Options{})
	req := httptest.NewRequest(http.MethodPost, "/api/tabs", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("empty origin without token: %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/tabs", nil)
	req.Header.Set("X-ToolDock-Token", CSRFToken)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code == http.StatusForbidden {
		t.Fatal("token should allow empty origin")
	}
}

func TestOriginAllowsLoopback(t *testing.T) {
	h := Handler(nil, Options{})
	req := hubRequest(http.MethodPost, "/api/tabs", nil)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code == http.StatusForbidden {
		t.Fatal("loopback origin should be allowed")
	}

	req = httptest.NewRequest(http.MethodGet, "/api/health", nil)
	req.Header.Set("Origin", "https://evil.example")
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET should not be origin-gated: %d", rr.Code)
	}
}

func TestDeleteTraversalID(t *testing.T) {
	h := Handler(nil, Options{})
	req := hubRequest(http.MethodDelete, "/api/tools/..%2fetc", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status %d body=%s", rr.Code, rr.Body.String())
	}
}
