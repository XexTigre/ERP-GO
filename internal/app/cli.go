package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"encoding/json"
	"github.com/XexTigre/ERP-GO/internal/catalog"
	"github.com/XexTigre/ERP-GO/internal/identity"
	"github.com/XexTigre/ERP-GO/internal/inventory"
	"github.com/XexTigre/ERP-GO/internal/platform/config"
	"github.com/XexTigre/ERP-GO/internal/platform/db"
	"github.com/XexTigre/ERP-GO/internal/platform/validate"
)

func Main(adminOnly bool) {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	if err := run(logger, adminOnly); err != nil {
		logger.Error("execucao encerrada", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger, adminOnly bool) error {
	command := "serve"
	args := os.Args[1:]
	if len(args) > 0 {
		command = args[0]
		args = args[1:]
	}
	if adminOnly && command == "serve" {
		return errors.New("use: erpctl migrate | bootstrap | key | revoke-key | verify-ledger")
	}
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	var name, tenant, role, keyID string
	switch command {
	case "bootstrap":
		flags.StringVar(&name, "name", "", "nome da empresa")
	case "verify-ledger":
		flags.StringVar(&tenant, "tenant", "", "ID da empresa")
	case "key":
		flags.StringVar(&tenant, "tenant", "", "ID da empresa")
		flags.StringVar(&role, "role", "reader", "admin, cataloger, stock ou reader")
	case "revoke-key":
		flags.StringVar(&keyID, "id", "", "ID da credencial")
	case "serve", "migrate":
	default:
		return errors.New("use: erp serve | migrate | bootstrap -name NOME | key -tenant ID -role PERFIL | revoke-key -id ID")
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("argumentos inesperados")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	connect, cancel := context.WithTimeout(ctx, 10*time.Second)
	cfg, err := config.Load()
	if err != nil {
		cancel()
		return err
	}
	pool, err := db.Open(connect, cfg.DatabaseURL, cfg.MaxConns)
	cancel()
	if err != nil {
		return err
	}
	defer pool.Close()
	store := &identity.PostgresRepository{Pool: pool}
	stock := &inventory.PostgresRepository{Pool: pool}
	if command != "serve" {
		operation, done := context.WithTimeout(ctx, 60*time.Second)
		defer done()
		switch command {
		case "migrate":
			return db.Migrate(operation, pool)
		case "bootstrap":
			id, raw, err := store.Bootstrap(operation, name)
			if err != nil {
				return err
			}
			fmt.Printf("tenant_id=%s\napi_token=%s\nGuarde esta credencial: ela so e exibida agora.\n", id, raw)
		case "key":
			id, raw, err := store.IssueKey(operation, tenant, identity.Role(role))
			if err != nil {
				return err
			}
			fmt.Printf("key_id=%s\napi_token=%s\nGuarde esta credencial: ela so e exibida agora.\n", id, raw)
		case "revoke-key":
			return store.RevokeKey(operation, keyID)
		case "verify-ledger":
			if !validate.ValidID(tenant) {
				return errors.New("-tenant deve ser um UUID válido")
			}
			items, err := stock.VerifyLedger(operation, tenant)
			if err != nil {
				return err
			}
			if err = json.NewEncoder(os.Stdout).Encode(items); err != nil {
				return err
			}
			if len(items) > 0 {
				return errors.New("razão e saldo divergentes")
			}

		}
		return nil
	}
	services := Services{Identity: identity.New(store), Catalog: catalog.New(&catalog.PostgresRepository{Pool: pool}), Inventory: inventory.New(stock), Ready: pool.Ping}
	server := &http.Server{Addr: cfg.HTTPAddr, Handler: NewHandler(services, logger), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 * 1024}

	failed := make(chan error, 1)
	go func() { logger.Info("API iniciada", "address", cfg.HTTPAddr); failed <- server.ListenAndServe() }()
	select {
	case err = <-failed:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(shutdown)
	}
}
