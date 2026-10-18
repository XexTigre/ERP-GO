package main

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

	"github.com/XexTigre/ERP-GO/internal/httpapi"
	"github.com/XexTigre/ERP-GO/internal/postgres"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	if err := run(logger); err != nil {
		logger.Error("execucao encerrada", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	command := "serve"
	args := os.Args[1:]
	if len(args) > 0 {
		command = args[0]
		args = args[1:]
	}
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	var name, tenant, role, keyID string
	switch command {
	case "bootstrap":
		flags.StringVar(&name, "name", "", "nome da empresa")
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
	store, err := postgres.Open(connect, os.Getenv("DATABASE_URL"))
	cancel()
	if err != nil {
		return err
	}
	defer store.Pool.Close()
	if command != "serve" {
		operation, done := context.WithTimeout(ctx, 60*time.Second)
		defer done()
		switch command {
		case "migrate":
			return store.Migrate(operation)
		case "bootstrap":
			id, raw, err := store.Bootstrap(operation, name)
			if err != nil {
				return err
			}
			fmt.Printf("tenant_id=%s\napi_token=%s\nGuarde esta credencial: ela so e exibida agora.\n", id, raw)
		case "key":
			id, raw, err := store.IssueKey(operation, tenant, role)
			if err != nil {
				return err
			}
			fmt.Printf("key_id=%s\napi_token=%s\nGuarde esta credencial: ela so e exibida agora.\n", id, raw)
		case "revoke-key":
			return store.RevokeKey(operation, keyID)
		}
		return nil
	}
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8080"
	}
	server := &http.Server{Addr: addr, Handler: httpapi.New(store, logger), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 * 1024}
	failed := make(chan error, 1)
	go func() { logger.Info("API iniciada", "address", addr); failed <- server.ListenAndServe() }()
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
