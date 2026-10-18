package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/XexTigre/ERP-GO/internal/erp"
	"github.com/XexTigre/ERP-GO/internal/postgres"
)

type API struct {
	store  *postgres.Store
	logger *slog.Logger
}
type endpoint func(http.ResponseWriter, *http.Request, erp.Principal)

func New(store *postgres.Store, logger *slog.Logger) http.Handler {
	a := &API{store: store, logger: logger}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]string{"status": "ok"}) })
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := store.Pool.Ping(ctx); err != nil {
			fail(w, 503, "banco indisponivel")
			return
		}
		writeJSON(w, 200, map[string]string{"status": "ready"})
	})
	mux.HandleFunc("GET /v1/brands", a.protect("", a.brands))
	mux.HandleFunc("POST /v1/brands", a.protect("catalog", a.createBrand))
	mux.HandleFunc("GET /v1/branches", a.protect("", a.branches))
	mux.HandleFunc("POST /v1/branches", a.protect("admin", a.createBranch))
	mux.HandleFunc("GET /v1/products", a.protect("", a.products))
	mux.HandleFunc("POST /v1/products", a.protect("catalog", a.createProduct))
	mux.HandleFunc("GET /v1/products/{id}", a.protect("", a.product))
	mux.HandleFunc("POST /v1/movements", a.protect("stock", a.move))
	mux.HandleFunc("GET /v1/movements/{id}", a.protect("", a.movement))
	mux.HandleFunc("GET /v1/balances", a.protect("", a.balances))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		mux.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a *API) protect(resource string, next endpoint) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		kind, raw, ok := strings.Cut(header, " ")
		if !ok || !strings.EqualFold(kind, "Bearer") || len(raw) != 64 {
			w.Header().Set("WWW-Authenticate", "Bearer")
			fail(w, 401, "credencial invalida")
			return
		}
		p, err := a.store.Authenticate(r.Context(), raw)
		if errors.Is(err, erp.ErrNotFound) {
			w.Header().Set("WWW-Authenticate", "Bearer")
			fail(w, 401, "credencial invalida")
			return
		}
		if err != nil {
			a.logger.Error("autenticacao indisponivel", "error", err)
			fail(w, 503, "servico indisponivel")
			return
		}
		if resource != "" && !p.CanWrite(resource) {
			fail(w, 403, "sem permissao")
			return
		}
		next(w, r, p)
	}
}

func decode(w http.ResponseWriter, r *http.Request, out interface{}) bool {
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/json" {
		fail(w, 415, "use Content-Type application/json")
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err = dec.Decode(out); err != nil {
		fail(w, 400, "JSON invalido ou acima do limite de 64 KiB")
		return false
	}
	if err = dec.Decode(new(interface{})); err != io.EOF {
		fail(w, 400, "envie apenas um objeto JSON")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func fail(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func (a *API) result(w http.ResponseWriter, status int, v interface{}, err error) {
	if err == nil {
		writeJSON(w, status, v)
		return
	}
	switch {
	case errors.Is(err, erp.ErrInvalid):
		fail(w, 400, "dados invalidos")
	case errors.Is(err, erp.ErrNotFound):
		fail(w, 404, "registro nao encontrado")
	case errors.Is(err, erp.ErrConflict):
		fail(w, 409, "cadastro duplicado ou chave reutilizada com outros dados")
	case errors.Is(err, erp.ErrInsufficient):
		fail(w, 409, "saldo insuficiente ou limite de saldo excedido")
	default:
		a.logger.Error("falha na operacao", "error", err)
		fail(w, 500, "falha interna")
	}
}

func paging(w http.ResponseWriter, r *http.Request) (string, int, bool) {
	after := r.URL.Query().Get("after")
	if after != "" && !erp.ValidID(after) {
		fail(w, 400, "after invalido")
		return "", 0, false
	}
	limit := 30
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 100 {
			fail(w, 400, "limit deve estar entre 1 e 100")
			return "", 0, false
		}
		limit = n
	}
	return after, limit, true
}

type page[T any] struct {
	Items      []T    `json:"items"`
	NextCursor string `json:"next_cursor,omitempty"`
}

func paged[T any](items []T, limit int, id func(T) string) page[T] {
	p := page[T]{Items: items}
	if len(items) > limit {
		p.Items = items[:limit]
		p.NextCursor = id(items[limit-1])
	}
	return p
}

func (a *API) brands(w http.ResponseWriter, r *http.Request, p erp.Principal) {
	after, limit, ok := paging(w, r)
	if !ok {
		return
	}
	items, err := a.store.Brands(r.Context(), p, after, limit+1)
	a.result(w, 200, paged(items, limit, func(v erp.Brand) string { return v.ID }), err)
}

func (a *API) createBrand(w http.ResponseWriter, r *http.Request, p erp.Principal) {
	var in struct {
		Name string `json:"name"`
	}
	if !decode(w, r, &in) {
		return
	}
	v, err := a.store.CreateBrand(r.Context(), p, in.Name)
	a.result(w, 201, v, err)
}

func (a *API) branches(w http.ResponseWriter, r *http.Request, p erp.Principal) {
	after, limit, ok := paging(w, r)
	if !ok {
		return
	}
	items, err := a.store.Branches(r.Context(), p, after, limit+1)
	a.result(w, 200, paged(items, limit, func(v erp.Branch) string { return v.ID }), err)
}

func (a *API) createBranch(w http.ResponseWriter, r *http.Request, p erp.Principal) {
	var in struct {
		Name string `json:"name"`
	}
	if !decode(w, r, &in) {
		return
	}
	v, err := a.store.CreateBranch(r.Context(), p, in.Name)
	a.result(w, 201, v, err)
}

func (a *API) products(w http.ResponseWriter, r *http.Request, p erp.Principal) {
	after, limit, ok := paging(w, r)
	if !ok {
		return
	}
	q := r.URL.Query().Get("q")
	if !utf8.ValidString(q) || utf8.RuneCountInString(q) > 120 || strings.ContainsRune(q, 0) {
		fail(w, 400, "q invalido")
		return
	}
	items, err := a.store.Products(r.Context(), p, q, after, limit+1)
	a.result(w, 200, paged(items, limit, func(v erp.Product) string { return v.ID }), err)
}

func (a *API) createProduct(w http.ResponseWriter, r *http.Request, p erp.Principal) {
	var in erp.ProductInput
	if !decode(w, r, &in) {
		return
	}
	v, err := a.store.CreateProduct(r.Context(), p, in)
	a.result(w, 201, v, err)
}

func (a *API) product(w http.ResponseWriter, r *http.Request, p erp.Principal) {
	v, err := a.store.Product(r.Context(), p, r.PathValue("id"))
	a.result(w, 200, v, err)
}

func (a *API) move(w http.ResponseWriter, r *http.Request, p erp.Principal) {
	var in erp.MovementInput
	if !decode(w, r, &in) {
		return
	}
	v, repeated, err := a.store.Move(r.Context(), p, r.Header.Get("Idempotency-Key"), in)
	status := 201
	if repeated {
		status = 200
	}
	a.result(w, status, v, err)
}

func (a *API) movement(w http.ResponseWriter, r *http.Request, p erp.Principal) {
	v, err := a.store.Movement(r.Context(), p, r.PathValue("id"))
	a.result(w, 200, v, err)
}

func (a *API) balances(w http.ResponseWriter, r *http.Request, p erp.Principal) {
	after, limit, ok := paging(w, r)
	if !ok {
		return
	}
	branch := r.URL.Query().Get("branch_id")
	if !erp.ValidID(branch) {
		fail(w, 400, "branch_id obrigatorio e valido")
		return
	}
	items, err := a.store.Balances(r.Context(), p, branch, after, limit+1)
	a.result(w, 200, paged(items, limit, func(v erp.Balance) string { return v.ProductID }), err)
}
