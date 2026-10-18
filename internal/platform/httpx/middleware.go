package httpx

import (
	"context"
	"crypto/rand"
	"log/slog"
	"net/http"
	"time"
)

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	if w.status != 0 {
		return
	}
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}
func (w *statusWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(200)
	}
	return w.ResponseWriter.Write(p)
}
func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func Middleware(next http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestID := rand.Text()
		w.Header().Set("X-Request-ID", requestID)
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		sw := &statusWriter{ResponseWriter: w}
		defer func() {
			if v := recover(); v != nil {
				logger.ErrorContext(r.Context(), "panic em requisição", "request_id", requestID, "panic_type", panicType(v))
				if sw.status == 0 {
					Fail(sw, r, 500, "internal_error", "Falha interna")
				}
			}
			status := sw.status
			if status == 0 {
				status = 200
			}
			logger.InfoContext(r.Context(), "http_request", "request_id", requestID, "method", r.Method, "path", r.URL.Path, "status", status, "duration_ms", time.Since(start).Milliseconds())
		}()
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		next.ServeHTTP(sw, r.WithContext(ctx))
	})
}

// Panic values can contain secrets; only their category is logged.
func panicType(v any) string {
	switch v.(type) {
	case error:
		return "error"
	case string:
		return "string"
	default:
		return "other"
	}
}
