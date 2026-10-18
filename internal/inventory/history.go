package inventory

import (
	"context"
	"encoding/base64"
	"strings"
	"time"

	"github.com/XexTigre/ERP-GO/internal/identity"
	"github.com/XexTigre/ERP-GO/internal/platform/db"
	"github.com/XexTigre/ERP-GO/internal/platform/fault"
	"github.com/XexTigre/ERP-GO/internal/platform/validate"
)

type MovementFilter struct {
	BranchID, ProductID, Kind, After string
	Limit                            int
}
type cursor struct {
	at time.Time
	id string
}

func parseCursor(raw string) (cursor, error) {
	if raw == "" {
		return cursor{}, nil
	}
	if len(raw) > 180 {
		return cursor{}, fault.ErrInvalid
	}
	data, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return cursor{}, fault.ErrInvalid
	}
	at, id, ok := strings.Cut(string(data), "/")
	t, err := time.Parse(time.RFC3339Nano, at)
	if !ok || err != nil || !validate.ValidID(id) {
		return cursor{}, fault.ErrInvalid
	}
	return cursor{at: t, id: strings.ToLower(id)}, nil
}
func MovementCursor(m Movement) string {
	return base64.RawURLEncoding.EncodeToString([]byte(m.CreatedAt.UTC().Format(time.RFC3339Nano) + "/" + m.ID))
}
func (f MovementFilter) Validate() error {
	for _, id := range []string{f.BranchID, f.ProductID} {
		if id != "" && !validate.ValidID(id) {
			return fault.ErrInvalid
		}
	}
	if f.Kind != "" && f.Kind != "entrada" && f.Kind != "saida" {
		return fault.ErrInvalid
	}
	if f.Limit < 1 || f.Limit > 101 {
		return fault.ErrInvalid
	}
	_, err := parseCursor(f.After)
	return err
}
func (s *PostgresRepository) Movements(ctx context.Context, p identity.Principal, f MovementFilter) ([]Movement, error) {
	if err := f.Validate(); err != nil {
		return nil, err
	}
	c, _ := parseCursor(f.After)
	var before any
	if !c.at.IsZero() {
		before = c.at
	}
	rows, err := db.ForTenant(s.Pool, p.TenantID).Query(ctx, `SELECT id,branch_id,product_id,kind,quantity,reason,idempotency_key,created_by,created_at,balance_after
 FROM stock_movements WHERE tenant_id=$1 AND ($2::uuid IS NULL OR branch_id=$2)
 AND ($3::uuid IS NULL OR product_id=$3) AND ($4='' OR kind=$4)
 AND ($5::timestamptz IS NULL OR (created_at,id)<($5,$6::uuid))
 ORDER BY created_at DESC,id DESC LIMIT $7`, p.TenantID, db.NullID(f.BranchID), db.NullID(f.ProductID), f.Kind, before, db.NullID(c.id), f.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Movement{}
	for rows.Next() {
		var v Movement
		if err = rows.Scan(&v.ID, &v.BranchID, &v.ProductID, &v.Kind, &v.Quantity, &v.Reason, &v.IdempotencyKey, &v.CreatedBy, &v.CreatedAt, &v.BalanceAfter); err != nil {
			return nil, err
		}
		items = append(items, v)
	}
	return items, rows.Err()
}

// ProductBalances includes every store in this tenant, including a zero balance
// without a previous movement. Inventory depends on IDs, not on catalog code.
func (s *PostgresRepository) ProductBalances(ctx context.Context, p identity.Principal, id, after string, limit int) ([]ProductBalance, error) {
	if !validate.ValidID(id) {
		return nil, fault.ErrInvalid
	}
	var exists bool
	if err := db.ForTenant(s.Pool, p.TenantID).QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM products WHERE tenant_id=$1 AND id=$2)`, p.TenantID, id).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, fault.ErrNotFound
	}
	rows, err := db.ForTenant(s.Pool, p.TenantID).Query(ctx, `SELECT b.id,b.name,COALESCE(s.quantity,0) FROM branches b
 LEFT JOIN stock_balances s ON s.tenant_id=b.tenant_id AND s.branch_id=b.id AND s.product_id=$2
 WHERE b.tenant_id=$1 AND ($3::uuid IS NULL OR b.id>$3) ORDER BY b.id LIMIT $4`, p.TenantID, id, db.NullID(after), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ProductBalance{}
	for rows.Next() {
		var v ProductBalance
		if err = rows.Scan(&v.BranchID, &v.Branch, &v.Quantity); err != nil {
			return nil, err
		}
		items = append(items, v)
	}
	return items, rows.Err()
}
