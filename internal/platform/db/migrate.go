package db

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/XexTigre/ERP-GO/migrations"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
)

func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(662210018)`); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (name text PRIMARY KEY, checksum text NOT NULL, applied_at timestamptz NOT NULL DEFAULT now())`); err != nil {
		return err
	}
	files, err := migrations.Files.ReadDir(".")
	if err != nil {
		return err
	}
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".sql") {
			continue
		}
		body, err := migrations.Files.ReadFile(file.Name())
		if err != nil {
			return err
		}
		digest := sha256.Sum256(body)
		checksum := hex.EncodeToString(digest[:])
		var existing string
		err = tx.QueryRow(ctx, `SELECT checksum FROM schema_migrations WHERE name=$1`, file.Name()).Scan(&existing)
		if err == nil {
			if existing != checksum {
				return fmt.Errorf("migracao alterada: %s", file.Name())
			}
			continue
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if _, err = tx.Exec(ctx, string(body)); err != nil {
			return fmt.Errorf("migracao %s: %w", file.Name(), err)
		}
		if _, err = tx.Exec(ctx, `INSERT INTO schema_migrations (name, checksum) VALUES ($1,$2)`, file.Name(), checksum); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
