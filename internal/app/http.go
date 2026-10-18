package app

import (
	"context"
	"github.com/XexTigre/ERP-GO/internal/catalog"
	"github.com/XexTigre/ERP-GO/internal/identity"
	"github.com/XexTigre/ERP-GO/internal/inventory"
	"github.com/XexTigre/ERP-GO/internal/platform/httpx"
	"github.com/XexTigre/ERP-GO/internal/web"
	"log/slog"
	"net/http"
	"time"
)

type Services struct {
	Identity  *identity.Service
	Catalog   *catalog.Service
	Inventory *inventory.Service
	Ready     func(context.Context) error
}

func NewHandler(s Services, logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	reply := httpx.Responder{Logger: logger}
	auth := identity.Middleware{Service: s.Identity, Logger: logger}
	identity.Register(mux, s.Identity, auth, reply)
	catalog.Register(mux, s.Catalog, auth, reply)
	inventory.Register(mux, s.Inventory, auth, reply)
	mux.Handle("GET /preview/v1/", http.StripPrefix("/preview/v1/", web.Preview()))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, 200, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if s.Ready == nil || s.Ready(ctx) != nil {
			httpx.Fail(w, r, 503, "unavailable", "Banco indisponível")
			return
		}
		httpx.WriteJSON(w, 200, map[string]string{"status": "ready"})
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		httpx.Fail(w, r, 404, "not_found", "Rota não encontrada")
	})
	return httpx.Middleware(mux, logger)
}
