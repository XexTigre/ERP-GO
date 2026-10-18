package postgres

import (
	"context"
	"strings"

	"github.com/XexTigre/ERP-GO/internal/erp"
)

func (s *Store) CreateBrand(ctx context.Context, p erp.Principal, name string) (erp.Brand, error) {
	var b erp.Brand
	name, err := erp.CleanText(name, 120)
	if err != nil {
		return b, err
	}
	err = s.Pool.QueryRow(ctx, `INSERT INTO brands (tenant_id,name,name_key) VALUES ($1,$2,$3) RETURNING id,name`, p.TenantID, name, erp.Normalize(name)).Scan(&b.ID, &b.Name)
	return b, classify(err)
}

func (s *Store) Brands(ctx context.Context, p erp.Principal, after string, limit int) ([]erp.Brand, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id,name FROM brands WHERE tenant_id=$1 AND ($2::uuid IS NULL OR id>$2::uuid) ORDER BY id LIMIT $3`, p.TenantID, nullID(after), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []erp.Brand{}
	for rows.Next() {
		var b erp.Brand
		if err = rows.Scan(&b.ID, &b.Name); err != nil {
			return nil, err
		}
		items = append(items, b)
	}
	return items, rows.Err()
}

func (s *Store) CreateBranch(ctx context.Context, p erp.Principal, name string) (erp.Branch, error) {
	var b erp.Branch
	name, err := erp.CleanText(name, 120)
	if err != nil {
		return b, err
	}
	err = s.Pool.QueryRow(ctx, `INSERT INTO branches (tenant_id,name,name_key) VALUES ($1,$2,$3) RETURNING id,name`, p.TenantID, name, erp.Normalize(name)).Scan(&b.ID, &b.Name)
	return b, classify(err)
}

func (s *Store) Branches(ctx context.Context, p erp.Principal, after string, limit int) ([]erp.Branch, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id,name FROM branches WHERE tenant_id=$1 AND ($2::uuid IS NULL OR id>$2::uuid) ORDER BY id LIMIT $3`, p.TenantID, nullID(after), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []erp.Branch{}
	for rows.Next() {
		var b erp.Branch
		if err = rows.Scan(&b.ID, &b.Name); err != nil {
			return nil, err
		}
		items = append(items, b)
	}
	return items, rows.Err()
}

func (s *Store) CreateProduct(ctx context.Context, p erp.Principal, in erp.ProductInput) (erp.Product, error) {
	var product erp.Product
	if err := in.Validate(); err != nil {
		return product, err
	}
	product.ProductInput = in
	err := s.Pool.QueryRow(ctx, `INSERT INTO products (tenant_id,brand_id,sku,sku_key,manufacturer_code,code_key,description) VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id,unit`, p.TenantID, in.BrandID, in.SKU, erp.Normalize(in.SKU), in.ManufacturerCode, erp.Normalize(in.ManufacturerCode), in.Description).Scan(&product.ID, &product.Unit)
	if err != nil {
		return product, classify(err)
	}
	return s.Product(ctx, p, product.ID)
}

func (s *Store) Product(ctx context.Context, p erp.Principal, id string) (erp.Product, error) {
	var v erp.Product
	if !erp.ValidID(id) {
		return v, erp.ErrInvalid
	}
	err := s.Pool.QueryRow(ctx, `SELECT p.id,p.brand_id,p.sku,p.manufacturer_code,p.description,p.unit,b.name FROM products p JOIN brands b ON b.tenant_id=p.tenant_id AND b.id=p.brand_id WHERE p.tenant_id=$1 AND p.id=$2`, p.TenantID, id).Scan(&v.ID, &v.BrandID, &v.SKU, &v.ManufacturerCode, &v.Description, &v.Unit, &v.Brand)
	return v, classify(err)
}

func (s *Store) Products(ctx context.Context, p erp.Principal, q, after string, limit int) ([]erp.Product, error) {
	// Escape LIKE metacharacters: searching for '%' means a literal percent.
	q = strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(strings.ToLower(strings.TrimSpace(q)))
	rows, err := s.Pool.Query(ctx, `SELECT p.id,p.brand_id,p.sku,p.manufacturer_code,p.description,p.unit,b.name FROM products p JOIN brands b ON b.tenant_id=p.tenant_id AND b.id=p.brand_id WHERE p.tenant_id=$1 AND ($2='' OR p.search_text LIKE '%' || $2 || '%') AND ($3::uuid IS NULL OR p.id>$3::uuid) ORDER BY p.id LIMIT $4`, p.TenantID, q, nullID(after), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []erp.Product{}
	for rows.Next() {
		var v erp.Product
		if err = rows.Scan(&v.ID, &v.BrandID, &v.SKU, &v.ManufacturerCode, &v.Description, &v.Unit, &v.Brand); err != nil {
			return nil, err
		}
		items = append(items, v)
	}
	return items, rows.Err()
}

func nullID(value string) interface{} {
	if value == "" {
		return nil
	}
	return value
}
