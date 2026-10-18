//go:build integration

package integration_test

import (
	"context"
	"github.com/XexTigre/ERP-GO/internal/catalog"
	"github.com/XexTigre/ERP-GO/internal/identity"
	"github.com/XexTigre/ERP-GO/internal/platform/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
	"testing"
)

func TestRLSWithNonOwnerRuntimeRole(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	tenantA, keyA, err := s.Bootstrap(ctx, "Empresa sintética RLS A")
	if err != nil {
		t.Fatal(err)
	}
	tenantB, keyB, err := s.Bootstrap(ctx, "Empresa sintética RLS B")
	if err != nil {
		t.Fatal(err)
	}
	pA, err := s.Authenticate(ctx, keyA)
	if err != nil {
		t.Fatal(err)
	}
	pB, err := s.Authenticate(ctx, keyB)
	if err != nil {
		t.Fatal(err)
	}
	cat := &catalog.PostgresRepository{Pool: s.Pool}
	for _, p := range []identity.Principal{pA, pB} {
		if _, err = cat.CreateBrand(ctx, p, "Marca sintética"); err != nil {
			t.Fatal(err)
		}
	}
	var schema string
	if err = s.Pool.QueryRow(ctx, `SELECT current_schema()`).Scan(&schema); err != nil {
		t.Fatal(err)
	}
	role := "runtime_" + strings.TrimPrefix(schema, "integration_")
	quoted := pgx.Identifier{role}.Sanitize()
	qs := pgx.Identifier{schema}.Sanitize()
	if _, err = s.Pool.Exec(ctx, "CREATE ROLE "+quoted+" NOLOGIN NOSUPERUSER NOBYPASSRLS"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, e := s.Pool.Exec(ctx, "DROP OWNED BY "+quoted); e != nil {
			t.Error(e)
		}
		if _, e := s.Pool.Exec(ctx, "DROP ROLE "+quoted); e != nil {
			t.Error(e)
		}
	})
	if _, err = s.Pool.Exec(ctx, "GRANT USAGE ON SCHEMA "+qs+" TO "+quoted); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Pool.Exec(ctx, "GRANT SELECT,INSERT ON ALL TABLES IN SCHEMA "+qs+" TO "+quoted); err != nil {
		t.Fatal(err)
	}
	cfg := s.Pool.Config().Copy()
	cfg.AfterConnect = func(ctx context.Context, c *pgx.Conn) error { _, e := c.Exec(ctx, "SET ROLE "+quoted); return e }
	runtime, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close()
	var count int
	if err = runtime.QueryRow(ctx, `SELECT count(*) FROM brands`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("sem tenant leu dados: %d %v", count, err)
	}
	for _, tenant := range []string{tenantA, tenantB} {
		if err = db.ForTenant(runtime, tenant).QueryRow(ctx, `SELECT count(*) FROM brands`).Scan(&count); err != nil || count != 1 {
			t.Fatalf("RLS sem WHERE falhou: %d %v", count, err)
		}
	}
	if err = runtime.QueryRow(ctx, `SELECT count(*) FROM brands`).Scan(&count); err != nil || count != 0 {
		t.Fatal("tenant permaneceu no pool")
	}
	auth := &identity.PostgresRepository{Pool: runtime}
	p, err := auth.Authenticate(ctx, keyA)
	if err != nil || p.TenantID != tenantA {
		t.Fatalf("autenticação RLS: %+v %v", p, err)
	}
	if _, err = db.ForTenant(runtime, tenantA).Exec(ctx, `INSERT INTO brands(tenant_id,name,name_key) VALUES($1,'Tentativa','TENTATIVA')`, tenantB); err == nil {
		t.Fatal("RLS permitiu escrever em outra empresa")
	}
	if _, err = runtime.Exec(ctx, `CREATE TABLE forbidden(id int)`); err == nil {
		t.Fatal("runtime recebeu DDL")
	}
}
