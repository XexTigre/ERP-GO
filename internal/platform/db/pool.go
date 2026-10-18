package db

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Open(ctx context.Context, url string, maxConns int32) (*pgxpool.Pool, error) {
	if url == "" {
		return nil, errors.New("DATABASE_URL obrigatoria")
	}
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, errors.New("DATABASE_URL invalida")
	}
	cfg.MaxConns = maxConns
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, errors.New("falha ao configurar banco")
	}
	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, errors.New("banco indisponivel")
	}
	return pool, nil
}
