package httpx

import (
	"encoding/json"
	"errors"
	"github.com/XexTigre/ERP-GO/internal/platform/fault"
	"github.com/XexTigre/ERP-GO/internal/platform/validate"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strconv"
)

type Problem struct {
	Type      string `json:"type"`
	Title     string `json:"title"`
	Status    int    `json:"status"`
	Detail    string `json:"detail"`
	Instance  string `json:"instance"`
	Code      string `json:"code"`
	RequestID string `json:"request_id,omitempty"`
	Error     string `json:"error"` // Compatibility with the original E0 clients.
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func Fail(w http.ResponseWriter, r *http.Request, status int, code, detail string) {
	p := Problem{Type: "urn:erp-go:problem:" + code, Title: http.StatusText(status), Status: status, Detail: detail, Instance: r.URL.Path, Code: code, RequestID: w.Header().Get("X-Request-ID"), Error: detail}
	w.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(p)
}
func Decode(w http.ResponseWriter, r *http.Request, out any) bool {
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/json" {
		Fail(w, r, 415, "unsupported_media_type", "Use Content-Type application/json")
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err = dec.Decode(out); err != nil {
		Fail(w, r, 400, "invalid_json", "JSON inválido ou acima do limite de 64 KiB")
		return false
	}
	if err = dec.Decode(new(any)); err != io.EOF {
		Fail(w, r, 400, "invalid_json", "Envie apenas um objeto JSON")
		return false
	}
	return true
}

type Responder struct{ Logger *slog.Logger }

func (a Responder) Result(w http.ResponseWriter, r *http.Request, status int, v any, err error) {
	if err == nil {
		WriteJSON(w, status, v)
		return
	}
	code, detail, httpStatus := "internal_error", "Falha interna", 500
	switch {
	case errors.Is(err, fault.ErrInvalid):
		code, detail, httpStatus = "invalid_request", "Dados inválidos", 400
	case errors.Is(err, fault.ErrNotFound):
		code, detail, httpStatus = "not_found", "Registro não encontrado", 404
	case errors.Is(err, fault.ErrForbidden):
		code, detail, httpStatus = "forbidden", "Sem permissão", 403
	case errors.Is(err, fault.ErrConflict):
		code, detail, httpStatus = "duplicate_record", "Cadastro duplicado", 409
	default:
		a.Logger.ErrorContext(r.Context(), "falha na operação", "request_id", w.Header().Get("X-Request-ID"), "error", err)
	}
	var coded *fault.Coded
	if errors.As(err, &coded) {
		code, detail = coded.Code, coded.Message
	}
	Fail(w, r, httpStatus, code, detail)
}
func Paging(w http.ResponseWriter, r *http.Request) (string, int, bool) {
	after := r.URL.Query().Get("after")
	if after != "" && !validate.ValidID(after) {
		Fail(w, r, 400, "invalid_cursor", "after inválido")
		return "", 0, false
	}
	limit, ok := Limit(w, r)
	return after, limit, ok
}
func Limit(w http.ResponseWriter, r *http.Request) (int, bool) {
	limit := 30
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 100 {
			Fail(w, r, 400, "invalid_limit", "limit deve estar entre 1 e 100")
			return 0, false
		}
		limit = n
	}
	return limit, true
}

type Page[T any] struct {
	Items      []T    `json:"items"`
	NextCursor string `json:"next_cursor,omitempty"`
}

func Paged[T any](items []T, limit int, id func(T) string) Page[T] {
	p := Page[T]{Items: items}
	if p.Items == nil {
		p.Items = []T{}
	}
	if len(items) > limit {
		p.Items = items[:limit]
		p.NextCursor = id(items[limit-1])
	}
	return p
}
