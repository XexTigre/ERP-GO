package identity

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"github.com/XexTigre/ERP-GO/internal/platform/db"
	"github.com/XexTigre/ERP-GO/internal/platform/fault"
	"github.com/XexTigre/ERP-GO/internal/platform/validate"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
)

type PostgresRepository struct{ Pool *pgxpool.Pool }

func normalizeName(s string) string { return strings.ToUpper(strings.Join(strings.Fields(s), " ")) }
func token() (string, []byte, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", nil, err
	}
	plain := hex.EncodeToString(buf)
	digest := sha256.Sum256([]byte(plain))
	return plain, digest[:], nil
}

func (s *PostgresRepository) Bootstrap(ctx context.Context, name string) (string, string, error) {
	name, err := validate.CleanText(name, 120)
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

func (s *PostgresRepository) IssueKey(ctx context.Context, tenant string, role Role) (string, string, error) {
	if !validate.ValidID(tenant) || !role.Valid() {
		return "", "", fault.ErrInvalid
	}
	plain, digest, err := token()
	if err != nil {
		return "", "", err
	}
	var id string
	err = s.Pool.QueryRow(ctx, `INSERT INTO api_keys (tenant_id,token_hash,role) VALUES ($1,$2,$3) RETURNING id`, tenant, digest, role).Scan(&id)
	return id, plain, db.Classify(err)
}

func (s *PostgresRepository) RevokeKey(ctx context.Context, id string) error {
	if !validate.ValidID(id) {
		return fault.ErrInvalid
	}
	r, err := s.Pool.Exec(ctx, `UPDATE api_keys SET revoked_at=now() WHERE id=$1 AND revoked_at IS NULL`, id)
	if err != nil {
		return err
	}
	if r.RowsAffected() == 0 {
		return fault.ErrNotFound
	}
	return nil
}

func (s *PostgresRepository) Authenticate(ctx context.Context, raw string) (Principal, error) {
	var p Principal
	if len(raw) != 64 {
		return p, fault.ErrNotFound
	}
	digest := sha256.Sum256([]byte(raw))
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return p, err
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `SELECT set_config('app.key_hash',$1,true)`, hex.EncodeToString(digest[:])); err != nil {
		return p, err
	}
	err = tx.QueryRow(ctx, `SELECT tenant_id,id,role FROM api_keys WHERE token_hash=$1 AND revoked_at IS NULL`, digest[:]).Scan(&p.TenantID, &p.KeyID, &p.Role)
	if err == nil {
		err = tx.Commit(ctx)
	}
	return p, db.Classify(err)
}

func (s *PostgresRepository) CreateBranch(ctx context.Context, p Principal, name string) (Branch, error) {
	var b Branch
	name, err := validate.CleanText(name, 120)
	if err != nil {
		return b, err
	}
	err = db.ForTenant(s.Pool, p.TenantID).QueryRow(ctx, `INSERT INTO branches (tenant_id,name,name_key) VALUES ($1,$2,$3) RETURNING id,name`, p.TenantID, name, normalizeName(name)).Scan(&b.ID, &b.Name)
	return b, db.Classify(err)
}

func (s *PostgresRepository) Branches(ctx context.Context, p Principal, after string, limit int) ([]Branch, error) {
	rows, err := db.ForTenant(s.Pool, p.TenantID).Query(ctx, `SELECT id,name FROM branches WHERE tenant_id=$1 AND ($2::uuid IS NULL OR id>$2::uuid) ORDER BY id LIMIT $3`, p.TenantID, db.NullID(after), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Branch{}
	for rows.Next() {
		var b Branch
		if err = rows.Scan(&b.ID, &b.Name); err != nil {
			return nil, err
		}
		items = append(items, b)
	}
	return items, rows.Err()
}
