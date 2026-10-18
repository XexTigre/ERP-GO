package inventory

import (
	"github.com/XexTigre/ERP-GO/internal/identity"
	"github.com/XexTigre/ERP-GO/internal/platform/httpx"
	"github.com/XexTigre/ERP-GO/internal/platform/validate"
	"net/http"
)

type handler struct {
	service *Service
	reply   httpx.Responder
}

func Register(mux *http.ServeMux, s *Service, auth identity.Middleware, reply httpx.Responder) {
	a := &handler{s, reply}
	mux.HandleFunc("POST /v1/movements", auth.Protect("stock", a.move))
	mux.HandleFunc("GET /v1/movements/{id}", auth.Protect("", a.movement))
	mux.HandleFunc("GET /v1/balances", auth.Protect("", a.balances))
	mux.HandleFunc("GET /v1/movements", auth.Protect("", a.movements))
	mux.HandleFunc("GET /v1/products/{id}/balances", auth.Protect("", a.productBalances))
}
func (a *handler) move(w http.ResponseWriter, r *http.Request, p identity.Principal) {
	var in MovementInput
	if !httpx.Decode(w, r, &in) {
		return
	}
	v, repeated, err := a.service.Move(r.Context(), p, r.Header.Get("Idempotency-Key"), in)
	status := 201
	if repeated {
		status = 200
	}
	a.reply.Result(w, r, status, v, err)
}

func (a *handler) movement(w http.ResponseWriter, r *http.Request, p identity.Principal) {
	v, err := a.service.Movement(r.Context(), p, r.PathValue("id"))
	a.reply.Result(w, r, 200, v, err)
}

func (a *handler) balances(w http.ResponseWriter, r *http.Request, p identity.Principal) {
	after, limit, ok := httpx.Paging(w, r)
	if !ok {
		return
	}
	branch := r.URL.Query().Get("branch_id")
	if !validate.ValidID(branch) {
		httpx.Fail(w, r, 400, "invalid_branch", "branch_id obrigatório e válido")
		return
	}
	items, err := a.service.Balances(r.Context(), p, branch, after, limit+1)
	a.reply.Result(w, r, 200, httpx.Paged(items, limit, func(v Balance) string { return v.ProductID }), err)
}

func (a *handler) movements(w http.ResponseWriter, r *http.Request, p identity.Principal) {
	limit, ok := httpx.Limit(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	f := MovementFilter{BranchID: q.Get("branch_id"), ProductID: q.Get("product_id"), Kind: q.Get("kind"), After: q.Get("after"), Limit: limit + 1}
	if err := f.Validate(); err != nil {
		httpx.Fail(w, r, 400, "invalid_movement_filter", "Filtro ou cursor de histórico inválido")
		return
	}
	items, err := a.service.Movements(r.Context(), p, f)
	a.reply.Result(w, r, 200, httpx.Paged(items, limit, MovementCursor), err)
}
func (a *handler) productBalances(w http.ResponseWriter, r *http.Request, p identity.Principal) {
	after, limit, ok := httpx.Paging(w, r)
	if !ok {
		return
	}
	items, err := a.service.ProductBalances(r.Context(), p, r.PathValue("id"), after, limit+1)
	a.reply.Result(w, r, 200, httpx.Paged(items, limit, func(b ProductBalance) string { return b.BranchID }), err)
}
