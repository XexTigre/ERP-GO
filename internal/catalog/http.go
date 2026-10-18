package catalog

import (
	"github.com/XexTigre/ERP-GO/internal/identity"
	"github.com/XexTigre/ERP-GO/internal/platform/httpx"
	"net/http"
	"strings"
	"unicode/utf8"
)

type handler struct {
	service *Service
	reply   httpx.Responder
}

func Register(mux *http.ServeMux, s *Service, auth identity.Middleware, reply httpx.Responder) {
	a := &handler{s, reply}
	mux.HandleFunc("GET /v1/brands", auth.Protect("", a.brands))
	mux.HandleFunc("POST /v1/brands", auth.Protect("catalog", a.createBrand))
	mux.HandleFunc("GET /v1/products", auth.Protect("", a.products))
	mux.HandleFunc("POST /v1/products", auth.Protect("catalog", a.createProduct))
	mux.HandleFunc("GET /v1/products/{id}", auth.Protect("", a.product))
}
func (a *handler) brands(w http.ResponseWriter, r *http.Request, p identity.Principal) {
	after, limit, ok := httpx.Paging(w, r)
	if !ok {
		return
	}
	items, err := a.service.Brands(r.Context(), p, after, limit+1)
	a.reply.Result(w, r, 200, httpx.Paged(items, limit, func(v Brand) string { return v.ID }), err)
}

func (a *handler) createBrand(w http.ResponseWriter, r *http.Request, p identity.Principal) {
	var in struct {
		Name string `json:"name"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	v, err := a.service.CreateBrand(r.Context(), p, in.Name)
	a.reply.Result(w, r, 201, v, err)
}

func (a *handler) products(w http.ResponseWriter, r *http.Request, p identity.Principal) {
	after, limit, ok := httpx.Paging(w, r)
	if !ok {
		return
	}
	q := r.URL.Query().Get("q")
	if !utf8.ValidString(q) || utf8.RuneCountInString(q) > 120 || strings.ContainsRune(q, 0) {
		httpx.Fail(w, r, 400, "invalid_search", "q inválido")
		return
	}
	items, err := a.service.Products(r.Context(), p, q, after, limit+1)
	a.reply.Result(w, r, 200, httpx.Paged(items, limit, func(v Product) string { return v.ID }), err)
}

func (a *handler) createProduct(w http.ResponseWriter, r *http.Request, p identity.Principal) {
	var in ProductInput
	if !httpx.Decode(w, r, &in) {
		return
	}
	v, err := a.service.CreateProduct(r.Context(), p, in)
	a.reply.Result(w, r, 201, v, err)
}

func (a *handler) product(w http.ResponseWriter, r *http.Request, p identity.Principal) {
	v, err := a.service.Product(r.Context(), p, r.PathValue("id"))
	a.reply.Result(w, r, 200, v, err)
}
