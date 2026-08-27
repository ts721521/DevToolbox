package server

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"net/url"
	"strings"

	"github.com/ts721521/DevToolbox/internal/registry"
)

// CSRFToken is required on mutating requests that have no browser Origin
// (curl / old clients). Same-origin UI sends Origin: http://127.0.0.1:17890.
var CSRFToken = newCSRFToken()

func newCSRFToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("tooldock: crypto/rand unavailable: " + err.Error())
	}
	return hex.EncodeToString(b)
}

func withOrigin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			if !mutateOK(r) {
				http.Error(w, "forbidden origin", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func mutateOK(r *http.Request) bool {
	if !originOK(r) {
		return false
	}
	if strings.TrimSpace(r.Header.Get("Origin")) != "" {
		return true
	}
	got := strings.TrimSpace(r.Header.Get("X-ToolDock-Token"))
	return subtle.ConstantTimeCompare([]byte(got), []byte(CSRFToken)) == 1
}

func originOK(r *http.Request) bool {
	raw := strings.TrimSpace(r.Header.Get("Origin"))
	if raw == "" {
		return true
	}
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	if u.Scheme != "http" {
		return false
	}
	if !registry.LoopbackHost(u.Hostname()) {
		return false
	}
	return u.Port() == "17890"
}
