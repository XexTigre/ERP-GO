package httpx

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPanicReturnsProblemWithRequestIDWithoutSecret(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	h := Middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("secret-in-panic") }), logger)
	r := httptest.NewRequest("GET", "/v1/products?token=secret-in-query", nil)
	r.Header.Set("Authorization", "Bearer secret-in-header")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 500 || !strings.HasPrefix(w.Header().Get("Content-Type"), "application/problem+json") {
		t.Fatalf("resposta inesperada: %+v", w)
	}
	var problem Problem
	if err := json.Unmarshal(w.Body.Bytes(), &problem); err != nil {
		t.Fatal(err)
	}
	if problem.Code != "internal_error" || problem.RequestID == "" || problem.RequestID != w.Header().Get("X-Request-ID") {
		t.Fatalf("problema sem correlação: %+v", problem)
	}
	if strings.Contains(logs.String()+w.Body.String(), "secret-in-") {
		t.Fatal("segredo registrado")
	}
}
