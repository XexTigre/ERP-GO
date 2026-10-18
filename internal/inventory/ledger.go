package inventory

import (
	"context"
	"errors"

	"github.com/XexTigre/ERP-GO/internal/identity"
	"github.com/XexTigre/ERP-GO/internal/platform/db"
	"github.com/jackc/pgx/v5"
)

// Move serializes a request key, updates the projection, then appends the final
// journal entry. Neither a retry nor a failed transaction changes the journal.
func (s *PostgresRepository) Move(ctx context.Context, p identity.Principal, key string, in MovementInput) (Movement, bool, error) {
	var v Movement
	if err := in.Validate(key); err != nil {
		return v, false, err
	}
	tx, err := db.ForTenant(s.Pool, p.TenantID).Begin(ctx)
	if err != nil {
		return v, false, err
	}
	defer tx.Rollback(context.Background())
	// A hash collision only serializes unrelated requests; the unique constraint
	// and full tenant/key comparison still determine identity.
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, p.TenantID+":"+key); err != nil {
		return v, false, err
	}
	err = tx.QueryRow(ctx, `SELECT id,branch_id,product_id,kind,quantity,reason,idempotency_key,created_by,created_at,balance_after FROM stock_movements WHERE tenant_id=$1 AND idempotency_key=$2`, p.TenantID, key).Scan(&v.ID, &v.BranchID, &v.ProductID, &v.Kind, &v.Quantity, &v.Reason, &v.IdempotencyKey, &v.CreatedBy, &v.CreatedAt, &v.BalanceAfter)
	if err == nil {
		if v.MovementInput != in {
			return Movement{}, false, ErrIdempotencyConflict
		}
		return v, true, tx.Commit(ctx)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Movement{}, false, err
	}
	v = Movement{MovementInput: in, IdempotencyKey: key, CreatedBy: p.KeyID}
	_, err = tx.Exec(ctx, `INSERT INTO stock_balances (tenant_id,branch_id,product_id) VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`, p.TenantID, in.BranchID, in.ProductID)
	if err != nil {
		return v, false, db.Classify(err)
	}
	err = tx.QueryRow(ctx, `UPDATE stock_balances SET quantity=quantity+$4 WHERE tenant_id=$1 AND branch_id=$2 AND product_id=$3 AND quantity+$4 BETWEEN 0 AND 9000000000000 RETURNING quantity`, p.TenantID, in.BranchID, in.ProductID, in.Delta()).Scan(&v.BalanceAfter)
	if errors.Is(err, pgx.ErrNoRows) {
		return v, false, ErrInsufficient
	}
	if err != nil {
		return v, false, db.Classify(err)
	}
	err = tx.QueryRow(ctx, `INSERT INTO stock_movements (tenant_id,branch_id,product_id,kind,quantity,reason,idempotency_key,created_by,balance_after) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id,created_at`, p.TenantID, in.BranchID, in.ProductID, in.Kind, in.Quantity, in.Reason, key, p.KeyID, v.BalanceAfter).Scan(&v.ID, &v.CreatedAt)
	if err != nil {
		return v, false, db.Classify(err)
	}
	return v, false, tx.Commit(ctx)
}

// LedgerMismatch is diagnostic output; expected is a decimal integer string
// because SUM(bigint) is numeric in PostgreSQL.
type LedgerMismatch struct {
	BranchID  string `json:"branch_id"`
	ProductID string `json:"product_id"`
	Actual    int64  `json:"actual"`
	Expected  string `json:"expected"`
}

func (s *PostgresRepository) VerifyLedger(ctx context.Context, tenant string) ([]LedgerMismatch, error) {
	rows, err := db.ForTenant(s.Pool, tenant).Query(ctx, `WITH ledger AS (
 SELECT branch_id,product_id,sum(CASE kind WHEN 'entrada' THEN quantity ELSE -quantity END) AS expected
 FROM stock_movements WHERE tenant_id=$1 GROUP BY branch_id,product_id
 ), balances AS (SELECT branch_id,product_id,quantity FROM stock_balances WHERE tenant_id=$1)
 SELECT COALESCE(l.branch_id,b.branch_id),COALESCE(l.product_id,b.product_id),COALESCE(b.quantity,0),COALESCE(l.expected,0)::text
 FROM ledger l FULL JOIN balances b USING(branch_id,product_id)
 WHERE COALESCE(b.quantity,0)<>COALESCE(l.expected,0) ORDER BY 1,2`, tenant)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []LedgerMismatch{}
	for rows.Next() {
		var v LedgerMismatch
		if err = rows.Scan(&v.BranchID, &v.ProductID, &v.Actual, &v.Expected); err != nil {
			return nil, err
		}
		items = append(items, v)
	}
	return items, rows.Err()
}
