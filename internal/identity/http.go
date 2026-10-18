package identity

import (
	"errors"
	"github.com/XexTigre/ERP-GO/internal/platform/fault"
	"github.com/XexTigre/ERP-GO/internal/platform/httpx"
	"log/slog"
	"net/http"
	"strings"
)

type Endpoint func(http.ResponseWriter, *http.Request, Principal)
type Middleware struct {
	Service *Service
	Logger  *slog.Logger
}

func (a Middleware) Protect(resource Resource, next Endpoint) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		kind, raw, ok := strings.Cut(r.Header.Get("Authorization"), " ")
		if !ok || !strings.EqualFold(kind, "Bearer") || len(raw) != 64 {
			w.Header().Set("WWW-Authenticate", "Bearer")
			httpx.Fail(w, r, 401, "invalid_credential", "Credencial inválida")
			return
		}
		p, err := a.Service.Authenticate(r.Context(), raw)
		if errors.Is(err, fault.ErrNotFound) {
			w.Header().Set("WWW-Authenticate", "Bearer")
			httpx.Fail(w, r, 401, "invalid_credential", "Credencial inválida")
			return
		}
		if err != nil {
			a.Logger.ErrorContext(r.Context(), "autenticação indisponível", "error", err)
			httpx.Fail(w, r, 503, "unavailable", "Serviço indisponível")
			return
		}
		if resource != "" && !p.CanWrite(resource) {
			httpx.Fail(w, r, 403, "forbidden", "Sem permissão")
			return
		}
		next(w, r, p)
	}
}
func Register(mux *http.ServeMux, s *Service, auth Middleware, reply httpx.Responder) {
	mux.HandleFunc("GET /v1/branches", auth.Protect("", func(w http.ResponseWriter, r *http.Request, p Principal) {
		after, limit, ok := httpx.Paging(w, r)
		if !ok {
			return
		}
		items, err := s.Branches(r.Context(), p, after, limit+1)
		reply.Result(w, r, 200, httpx.Paged(items, limit, func(b Branch) string { return b.ID }), err)
	}))
	mux.HandleFunc("POST /v1/branches", auth.Protect(ResourceAdmin, func(w http.ResponseWriter, r *http.Request, p Principal) {
		var in struct {
			Name string `json:"name"`
		}
		if !httpx.Decode(w, r, &in) {
			return
		}
		v, err := s.CreateBranch(r.Context(), p, in.Name)
		reply.Result(w, r, 201, v, err)
	}))
}
