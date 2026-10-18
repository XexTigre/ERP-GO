//go:build integration

package httpapi_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/XexTigre/ERP-GO/internal/erp"
	"github.com/XexTigre/ERP-GO/internal/httpapi"
	"github.com/XexTigre/ERP-GO/internal/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func testStore(t *testing.T) *postgres.Store {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Fatal("TEST_DATABASE_URL obrigatoria com -tags=integration")
	}
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(cfg.ConnConfig.Database, "_test") {
		t.Fatal("use banco exclusivo com sufixo _test")
	}
	ctx := context.Background()
	admin, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 8)
	if _, err = rand.Read(buf); err != nil {
		t.Fatal(err)
	}
	schema := "integration_" + hex.EncodeToString(buf)
	quoted := pgx.Identifier{schema}.Sanitize()
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+quoted); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema + ",public"
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	store := &postgres.Store{Pool: pool}
	t.Cleanup(func() {
		pool.Close()
		if _, err := admin.Exec(context.Background(), "DROP SCHEMA "+quoted+" CASCADE"); err != nil {
			t.Error(err)
		}
		admin.Close()
	})
	if err = store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err = store.Migrate(ctx); err != nil {
		t.Fatalf("migracao nao idempotente: %v", err)
	}
	return store
}

func TestCatalogStockAndTenantBoundaries(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	tenantA, keyA, err := s.Bootstrap(ctx, "Empresa sintetica A")
	if err != nil {
		t.Fatal(err)
	}
	_, keyB, err := s.Bootstrap(ctx, "Empresa sintetica B")
	if err != nil {
		t.Fatal(err)
	}
	pA, err := s.Authenticate(ctx, keyA)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(httpapi.New(s, slog.New(slog.NewTextHandler(io.Discard, nil))))
	defer server.Close()
	call := func(token, method, path, key string, body interface{}, want int) json.RawMessage {
		t.Helper()
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		req, err := http.NewRequest(method, server.URL+path, bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		if key != "" {
			req.Header.Set("Idempotency-Key", key)
		}
		resp, err := server.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		payload, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != want {
			t.Fatalf("%s %s: %d, esperado %d: %s", method, path, resp.StatusCode, want, payload)
		}
		return payload
	}
	idOf := func(data []byte) string {
		t.Helper()
		var v struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(data, &v); err != nil || v.ID == "" {
			t.Fatalf("id ausente: %s", data)
		}
		return v.ID
	}
	branch := idOf(call(keyA, "POST", "/v1/branches", "", map[string]string{"name": "Loja sintetica 1"}, 201))
	branch2 := idOf(call(keyA, "POST", "/v1/branches", "", map[string]string{"name": "Loja sintetica 2"}, 201))
	brand := idOf(call(keyA, "POST", "/v1/brands", "", map[string]string{"name": "Fabricante teste A"}, 201))
	brand2 := idOf(call(keyA, "POST", "/v1/brands", "", map[string]string{"name": "Fabricante teste B"}, 201))
	call(keyA, "POST", "/v1/brands", "", map[string]string{"name": " fabricante   teste a "}, 409)
	input := erp.ProductInput{BrandID: brand, SKU: "TEST-001", ManufacturerCode: "001.24 BL", Description: "Peca sintetica 100% teste"}
	product := idOf(call(keyA, "POST", "/v1/products", "", input, 201))
	input.SKU = "TEST-002"
	call(keyA, "POST", "/v1/products", "", input, 409)
	input.BrandID = brand2
	input.Description = "Outra peca sintetica"
	call(keyA, "POST", "/v1/products", "", input, 201)
	input.SKU = "TEST-001"
	input.ManufacturerCode = "OUTRO"
	call(keyA, "POST", "/v1/products", "", input, 409)
	input.SKU = "TEST-003"
	input.ManufacturerCode = "001.24 AC"
	call(keyA, "POST", "/v1/products", "", input, 201)
	call(keyB, "GET", "/v1/products/"+product, "", nil, 404)
	call(keyB, "POST", "/v1/products", "", input, 404)
	call(keyB, "GET", "/v1/balances?branch_id="+branch, "", nil, 404)
	for _, role := range []string{"reader", "cataloger", "stock"} {
		_, key, err := s.IssueKey(ctx, tenantA, role)
		if err != nil {
			t.Fatal(err)
		}
		for _, path := range []string{"/v1/brands", "/v1/products", "/v1/branches", "/v1/movements"} {
			allowed := role == "cataloger" && (path == "/v1/brands" || path == "/v1/products") || role == "stock" && path == "/v1/movements"
			if !allowed {
				call(key, "POST", path, "", map[string]string{}, 403)
			}
		}
		call(key, "GET", "/v1/products", "", nil, 200)
	}
	for _, path := range []string{"/v1/products?after=invalid", "/v1/products?limit=0", "/v1/products?limit=101"} {
		call(keyA, "GET", path, "", nil, 400)
	}
	var page struct {
		Items  []erp.Product `json:"items"`
		Cursor string        `json:"next_cursor"`
	}
	json.Unmarshal(call(keyA, "GET", "/v1/products?limit=1", "", nil, 200), &page)
	if len(page.Items) != 1 || page.Cursor == "" {
		t.Fatal("paginacao ausente")
	}
	first := page.Items[0].ID
	page.Items = nil
	json.Unmarshal(call(keyA, "GET", "/v1/products?limit=1&after="+page.Cursor, "", nil, 200), &page)
	if len(page.Items) != 1 || page.Items[0].ID == first {
		t.Fatal("cursor repetiu produto")
	}
	json.Unmarshal(call(keyA, "GET", "/v1/products?q=%25", "", nil, 200), &page)
	if len(page.Items) != 1 {
		t.Fatalf("busca literal incorreta: %d", len(page.Items))
	}
	var foreignPage struct {
		Items []erp.Product `json:"items"`
	}
	if err := json.Unmarshal(call(keyB, "GET", "/v1/products", "", nil, 200), &foreignPage); err != nil {
		t.Fatal(err)
	}
	if len(foreignPage.Items) != 0 {
		t.Fatal("lista vazou produtos de outra empresa")
	}
	move := erp.MovementInput{BranchID: branch, ProductID: product, Kind: "entrada", Quantity: 10, Reason: "Entrada sintetica"}
	var original erp.Movement
	json.Unmarshal(call(keyA, "POST", "/v1/movements", "entry_0000000000001", move, 201), &original)
	var repeated erp.Movement
	json.Unmarshal(call(keyA, "POST", "/v1/movements", "entry_0000000000001", move, 200), &repeated)
	if original.ID != repeated.ID || repeated.BalanceAfter != 10 {
		t.Fatal("repeticao alterou entrada")
	}
	move.ProductID = strings.ToUpper(product)
	call(keyA, "POST", "/v1/movements", "entry_0000000000001", move, 200)
	move.ProductID = product
	move.Quantity = 11
	call(keyA, "POST", "/v1/movements", "entry_0000000000001", move, 409)
	call(keyA, "POST", "/v1/movements", "", move, 400)
	invalid := json.RawMessage(fmt.Sprintf(`{"branch_id":%q,"product_id":%q,"kind":"entrada","quantity":1.5,"reason":"Quantidade fracionada"}`, branch, product))
	call(keyA, "POST", "/v1/movements", "fractional_00000001", invalid, 400)
	call(keyB, "POST", "/v1/movements", "foreign_0000000001", move, 404)
	call(keyB, "GET", "/v1/movements/"+original.ID, "", nil, 404)
	move.Kind = "saida"
	call(keyA, "POST", "/v1/movements", "insufficient_000001", move, 409)
	var count int
	s.Pool.QueryRow(ctx, `SELECT count(*) FROM stock_movements WHERE tenant_id=$1`, tenantA).Scan(&count)
	if count != 1 {
		t.Fatal("saida recusada deixou movimento")
	}
	var empty struct {
		Items []erp.Balance `json:"items"`
	}
	json.Unmarshal(call(keyA, "GET", "/v1/balances?branch_id="+branch2, "", nil, 200), &empty)
	if len(empty.Items) != 0 {
		t.Fatal("saldo vazou entre lojas")
	}
	var wg sync.WaitGroup
	results := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			in := erp.MovementInput{BranchID: branch, ProductID: product, Kind: "saida", Quantity: 1, Reason: "Saida concorrente sintetica"}
			_, _, err := s.Move(ctx, pA, fmt.Sprintf("concurrent_%08d", i), in)
			results <- err
		}(i)
	}
	wg.Wait()
	close(results)
	success, insufficient := 0, 0
	for err := range results {
		switch {
		case err == nil:
			success++
		case errors.Is(err, erp.ErrInsufficient):
			insufficient++
		default:
			t.Fatal(err)
		}
	}
	if success != 10 || insufficient != 10 {
		t.Fatalf("overselling: sucesso=%d recusadas=%d", success, insufficient)
	}
	retries := make(chan bool, 12)
	failures := make(chan error, 12)
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			in := erp.MovementInput{BranchID: branch, ProductID: product, Kind: "entrada", Quantity: 1, Reason: "Retentativa sintetica"}
			_, repeated, err := s.Move(ctx, pA, "same_request_000001", in)
			retries <- repeated
			failures <- err
		}()
	}
	wg.Wait()
	close(retries)
	close(failures)
	newCount := 0
	for v := range retries {
		if !v {
			newCount++
		}
	}
	for err := range failures {
		if err != nil {
			t.Fatal(err)
		}
	}
	if newCount != 1 {
		t.Fatalf("pedido executado %d vezes", newCount)
	}
	balances, err := s.Balances(ctx, pA, branch, "", 100)
	if err != nil || len(balances) != 1 || balances[0].Quantity != 1 {
		t.Fatalf("saldo final incorreto: %+v, %v", balances, err)
	}
	var journalSum int64
	if err = s.Pool.QueryRow(ctx, `SELECT sum(CASE WHEN kind='entrada' THEN quantity ELSE -quantity END) FROM stock_movements WHERE tenant_id=$1`, tenantA).Scan(&journalSum); err != nil {
		t.Fatal(err)
	}
	if journalSum != 1 {
		t.Fatal("razao diverge do saldo")
	}
	if err = s.RevokeKey(ctx, pA.KeyID); err != nil {
		t.Fatal(err)
	}
	call(keyA, "GET", "/v1/products", "", nil, 401)
	if _, err = s.Pool.Exec(ctx, `UPDATE schema_migrations SET checksum='alterado'`); err != nil {
		t.Fatal(err)
	}
	if err = s.Migrate(ctx); err == nil {
		t.Fatal("migracao adulterada aceita")
	}
}
