package launcher

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPOKRejectsOffLoopbackRedirect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://example.com/", http.StatusFound)
	}))
	t.Cleanup(srv.Close)
	ok, _ := httpOK(srv.URL, "")
	if ok {
		t.Fatal("redirect off loopback must not count as healthy")
	}
}

func TestHTTPOKFollowsLoopbackRedirect(t *testing.T) {
	final := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	}))
	t.Cleanup(final.Close)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, final.URL, http.StatusFound)
	}))
	t.Cleanup(srv.Close)
	ok, _ := httpOK(srv.URL, "ready")
	if !ok {
		t.Fatal("loopback redirect with 2xx body should be healthy")
	}
}

func TestHTTPOKRejects4xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	ok, _ := httpOK(srv.URL, "")
	if ok {
		t.Fatal("404 must not count as healthy")
	}
}
