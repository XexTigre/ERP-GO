package db

import (
	"context"
	"github.com/XexTigre/ERP-GO/internal/platform/fault"
	"github.com/XexTigre/ERP-GO/internal/platform/validate"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TenantDB scopes every repository statement to a transaction-local tenant.
// A pooled connection never retains app.tenant_id after commit or rollback.
type TenantDB struct {
	pool   *pgxpool.Pool
	tenant string
}

func ForTenant(pool *pgxpool.Pool, tenant string) TenantDB { return TenantDB{pool, tenant} }
func (d TenantDB) Begin(ctx context.Context) (pgx.Tx, error) {
	if !validate.ValidID(d.tenant) {
		return nil, fault.ErrInvalid
	}
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `SELECT set_config('app.tenant_id',$1,true)`, d.tenant); err != nil {
		_ = tx.Rollback(context.Background())
		return nil, err
	}
	return tx, nil
}
func (d TenantDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	tx, err := d.Begin(ctx)
	if err != nil {
		return pgconn.CommandTag{}, err
	}
	defer tx.Rollback(context.Background())
	tag, err := tx.Exec(ctx, sql, args...)
	if err != nil {
		return tag, err
	}
	return tag, tx.Commit(ctx)
}

type scopedRow struct {
	db   TenantDB
	ctx  context.Context
	sql  string
	args []any
}

func (d TenantDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return scopedRow{d, ctx, sql, args}
}
func (r scopedRow) Scan(dest ...any) error {
	tx, err := r.db.Begin(r.ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	if err = tx.QueryRow(r.ctx, r.sql, r.args...).Scan(dest...); err != nil {
		return err
	}
	return tx.Commit(r.ctx)
}

type scopedRows struct {
	pgx.Rows
	tx     pgx.Tx
	ctx    context.Context
	closed bool
	err    error
}

func (d TenantDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	tx, err := d.Begin(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, sql, args...)
	if err != nil {
		_ = tx.Rollback(context.Background())
		return nil, err
	}
	return &scopedRows{Rows: rows, tx: tx, ctx: ctx}, nil
}
func (r *scopedRows) Close() {
	if r.closed {
		return
	}
	r.closed = true
	r.Rows.Close()
	if r.Rows.Err() != nil {
		r.err = r.Rows.Err()
		_ = r.tx.Rollback(context.Background())
		return
	}
	r.err = r.tx.Commit(r.ctx)
}
func (r *scopedRows) Next() bool {
	ok := r.Rows.Next()
	if !ok {
		r.Close()
	}
	return ok
}
func (r *scopedRows) Err() error {
	if r.err != nil {
		return r.err
	}
	return r.Rows.Err()
}
