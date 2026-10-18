package postgres

import (
	"context"
	"errors"

	"github.com/XexTigre/ERP-GO/internal/erp"
	"github.com/jackc/pgx/v5"
)

// Move commits the journal entry and its balance in the same transaction.
// A unique request key serializes retries; the UPDATE locks the balance row.
func (s *Store) Move(ctx context.Context, p erp.Principal, key string, in erp.MovementInput) (erp.Movement, bool, error) {
	var v erp.Movement
	if err := in.Validate(key); err != nil {
		return v, false, err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return v, false, err
	}
	defer tx.Rollback(context.Background())
	v.MovementInput = in
	v.IdempotencyKey = key
	v.CreatedBy = p.KeyID
	err = tx.QueryRow(ctx, `INSERT INTO stock_movements (tenant_id,branch_id,product_id,kind,quantity,reason,idempotency_key,created_by) VALUES ($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT (tenant_id,idempotency_key) DO NOTHING RETURNING id,created_at`, p.TenantID, in.BranchID, in.ProductID, in.Kind, in.Quantity, in.Reason, key, p.KeyID).Scan(&v.ID, &v.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		var existing erp.Movement
		err = tx.QueryRow(ctx, `SELECT id,branch_id,product_id,kind,quantity,reason,idempotency_key,created_by,created_at,balance_after FROM stock_movements WHERE tenant_id=$1 AND idempotency_key=$2`, p.TenantID, key).Scan(&existing.ID, &existing.BranchID, &existing.ProductID, &existing.Kind, &existing.Quantity, &existing.Reason, &existing.IdempotencyKey, &existing.CreatedBy, &existing.CreatedAt, &existing.BalanceAfter)
		if err != nil {
			return v, false, classify(err)
		}
		if existing.MovementInput != in {
			return v, false, erp.ErrConflict
		}
		return existing, true, tx.Commit(ctx)
	}
	if err != nil {
		return v, false, classify(err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO stock_balances (tenant_id,branch_id,product_id) VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`, p.TenantID, in.BranchID, in.ProductID)
	if err != nil {
		return v, false, classify(err)
	}
	err = tx.QueryRow(ctx, `UPDATE stock_balances SET quantity=quantity+$4 WHERE tenant_id=$1 AND branch_id=$2 AND product_id=$3 AND quantity+$4 BETWEEN 0 AND 9000000000000 RETURNING quantity`, p.TenantID, in.BranchID, in.ProductID, in.Delta()).Scan(&v.BalanceAfter)
	if errors.Is(err, pgx.ErrNoRows) {
		return v, false, erp.ErrInsufficient
	}
	if err != nil {
		return v, false, classify(err)
	}
	_, err = tx.Exec(ctx, `UPDATE stock_movements SET balance_after=$3 WHERE tenant_id=$1 AND id=$2`, p.TenantID, v.ID, v.BalanceAfter)
	if err != nil {
		return v, false, err
	}
	return v, false, tx.Commit(ctx)
}

func (s *Store) Balances(ctx context.Context, p erp.Principal, branch, after string, limit int) ([]erp.Balance, error) {
	var exists bool
	if err := s.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM branches WHERE tenant_id=$1 AND id=$2)`, p.TenantID, branch).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, erp.ErrNotFound
	}
	rows, err := s.Pool.Query(ctx, `SELECT p.id,p.sku,p.description,b.quantity FROM stock_balances b JOIN products p ON p.tenant_id=b.tenant_id AND p.id=b.product_id WHERE b.tenant_id=$1 AND b.branch_id=$2 AND ($3::uuid IS NULL OR p.id>$3::uuid) ORDER BY p.id LIMIT $4`, p.TenantID, branch, nullID(after), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []erp.Balance{}
	for rows.Next() {
		var v erp.Balance
		if err = rows.Scan(&v.ProductID, &v.SKU, &v.Description, &v.Quantity); err != nil {
			return nil, err
		}
		items = append(items, v)
	}
	return items, rows.Err()
}

func (s *Store) Movement(ctx context.Context, p erp.Principal, id string) (erp.Movement, error) {
	var v erp.Movement
	if !erp.ValidID(id) {
		return v, erp.ErrInvalid
	}
	err := s.Pool.QueryRow(ctx, `SELECT id,branch_id,product_id,kind,quantity,reason,idempotency_key,created_by,created_at,balance_after FROM stock_movements WHERE tenant_id=$1 AND id=$2`, p.TenantID, id).Scan(&v.ID, &v.BranchID, &v.ProductID, &v.Kind, &v.Quantity, &v.Reason, &v.IdempotencyKey, &v.CreatedBy, &v.CreatedAt, &v.BalanceAfter)
	return v, classify(err)
}
