package inventory

import (
	"context"
	"github.com/XexTigre/ERP-GO/internal/identity"
	"github.com/XexTigre/ERP-GO/internal/platform/db"
	"github.com/XexTigre/ERP-GO/internal/platform/fault"
	"github.com/XexTigre/ERP-GO/internal/platform/validate"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct{ Pool *pgxpool.Pool }

func (s *PostgresRepository) Balances(ctx context.Context, p identity.Principal, branch, after string, limit int) ([]Balance, error) {
	var exists bool
	if err := db.ForTenant(s.Pool, p.TenantID).QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM branches WHERE tenant_id=$1 AND id=$2)`, p.TenantID, branch).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, fault.ErrNotFound
	}
	rows, err := db.ForTenant(s.Pool, p.TenantID).Query(ctx, `SELECT p.id,p.sku,p.description,b.quantity FROM stock_balances b JOIN products p ON p.tenant_id=b.tenant_id AND p.id=b.product_id WHERE b.tenant_id=$1 AND b.branch_id=$2 AND ($3::uuid IS NULL OR p.id>$3::uuid) ORDER BY p.id LIMIT $4`, p.TenantID, branch, db.NullID(after), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Balance{}
	for rows.Next() {
		var v Balance
		if err = rows.Scan(&v.ProductID, &v.SKU, &v.Description, &v.Quantity); err != nil {
			return nil, err
		}
		items = append(items, v)
	}
	return items, rows.Err()
}

func (s *PostgresRepository) Movement(ctx context.Context, p identity.Principal, id string) (Movement, error) {
	var v Movement
	if !validate.ValidID(id) {
		return v, fault.ErrInvalid
	}
	err := db.ForTenant(s.Pool, p.TenantID).QueryRow(ctx, `SELECT id,branch_id,product_id,kind,quantity,reason,idempotency_key,created_by,created_at,balance_after FROM stock_movements WHERE tenant_id=$1 AND id=$2`, p.TenantID, id).Scan(&v.ID, &v.BranchID, &v.ProductID, &v.Kind, &v.Quantity, &v.Reason, &v.IdempotencyKey, &v.CreatedBy, &v.CreatedAt, &v.BalanceAfter)
	return v, db.Classify(err)
}
