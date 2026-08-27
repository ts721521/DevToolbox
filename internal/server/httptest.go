package server

import (
	"io"
	"net/http"
	"net/http/httptest"
)

func hubRequest(method, target string, body io.Reader) *http.Request {
	req := httptest.NewRequest(method, target, body)
	req.Header.Set("Origin", "http://127.0.0.1:17890")
	return req
}
