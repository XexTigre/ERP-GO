package app

import (
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAnonymousCannotMutate(t *testing.T) {
	handler := NewHandler(Services{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	for _, path := range []string{"/v1/brands", "/v1/branches", "/v1/products", "/v1/movements"} {
		request := httptest.NewRequest("POST", path, strings.NewReader(`{}`))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != 401 {
			t.Fatalf("%s: %d", path, response.Code)
		}
		if response.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("cache indevido")
		}
	}
}
