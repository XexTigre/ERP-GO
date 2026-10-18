package postgres

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/XexTigre/ERP-GO/internal/erp"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrations embed.FS

type Store struct{ Pool *pgxpool.Pool }

func Open(ctx context.Context, url string) (*Store, error) {
	if url == "" {
		return nil, errors.New("DATABASE_URL obrigatoria")
	}
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, errors.New("DATABASE_URL invalida")
	}
	cfg.MaxConns = 12
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, errors.New("falha ao configurar banco")
	}
	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, errors.New("banco indisponivel")
	}
	return &Store{Pool: pool}, nil
}

func (s *Store) Migrate(ctx context.Context) error {
	tx, err := s.Pool.Begin(ctx)
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
	files, err := migrations.ReadDir("migrations")
	if err != nil {
		return err
	}
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".sql") {
			continue
		}
		body, err := migrations.ReadFile("migrations/" + file.Name())
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

func token() (string, []byte, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", nil, err
	}
	plain := hex.EncodeToString(buf)
	digest := sha256.Sum256([]byte(plain))
	return plain, digest[:], nil
}

func (s *Store) Bootstrap(ctx context.Context, name string) (string, string, error) {
	name, err := erp.CleanText(name, 120)
	if err != nil {
		return "", "", err
	}
	plain, digest, err := token()
	if err != nil {
		return "", "", err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return "", "", err
	}
	defer tx.Rollback(context.Background())
	var id string
	if err = tx.QueryRow(ctx, `INSERT INTO tenants (name) VALUES ($1) RETURNING id`, name).Scan(&id); err != nil {
		return "", "", err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO api_keys (tenant_id,token_hash,role) VALUES ($1,$2,'admin')`, id, digest); err != nil {
		return "", "", err
	}
	err = tx.Commit(ctx)
	return id, plain, err
}

func (s *Store) IssueKey(ctx context.Context, tenant, role string) (string, string, error) {
	if !erp.ValidID(tenant) || (role != "admin" && role != "cataloger" && role != "stock" && role != "reader") {
		return "", "", erp.ErrInvalid
	}
	plain, digest, err := token()
	if err != nil {
		return "", "", err
	}
	var id string
	err = s.Pool.QueryRow(ctx, `INSERT INTO api_keys (tenant_id,token_hash,role) VALUES ($1,$2,$3) RETURNING id`, tenant, digest, role).Scan(&id)
	return id, plain, classify(err)
}

func (s *Store) RevokeKey(ctx context.Context, id string) error {
	if !erp.ValidID(id) {
		return erp.ErrInvalid
	}
	r, err := s.Pool.Exec(ctx, `UPDATE api_keys SET revoked_at=now() WHERE id=$1 AND revoked_at IS NULL`, id)
	if err != nil {
		return err
	}
	if r.RowsAffected() == 0 {
		return erp.ErrNotFound
	}
	return nil
}

func (s *Store) Authenticate(ctx context.Context, raw string) (erp.Principal, error) {
	var p erp.Principal
	if len(raw) != 64 {
		return p, erp.ErrNotFound
	}
	digest := sha256.Sum256([]byte(raw))
	err := s.Pool.QueryRow(ctx, `SELECT tenant_id,id,role FROM api_keys WHERE token_hash=$1 AND revoked_at IS NULL`, digest[:]).Scan(&p.TenantID, &p.KeyID, &p.Role)
	return p, classify(err)
}

func classify(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return erp.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return erp.ErrConflict
		case "23503":
			return erp.ErrNotFound
		case "23514", "22001", "22P02":
			return erp.ErrInvalid
		}
	}
	return err
}
