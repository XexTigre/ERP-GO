package catalog

import (
	"context"
	"errors"
	"github.com/XexTigre/ERP-GO/internal/identity"
	"github.com/XexTigre/ERP-GO/internal/platform/db"
	"github.com/XexTigre/ERP-GO/internal/platform/fault"
	"github.com/XexTigre/ERP-GO/internal/platform/validate"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
)

type PostgresRepository struct{ Pool *pgxpool.Pool }

func classify(err error) error {
	var p *pgconn.PgError
	if errors.As(err, &p) && p.Code == "23505" {
		switch p.ConstraintName {
		case "products_tenant_id_brand_id_code_key_key":
			return ErrDuplicateIdentity
		case "products_tenant_id_sku_key_key":
			return ErrDuplicateSKU
		}
	}
	return db.Classify(err)
}
func (s *PostgresRepository) CreateBrand(ctx context.Context, p identity.Principal, name string) (Brand, error) {
	var b Brand
	name, err := validate.CleanText(name, 120)
	if err != nil {
		return b, err
	}
	err = db.ForTenant(s.Pool, p.TenantID).QueryRow(ctx, `INSERT INTO brands (tenant_id,name,name_key) VALUES ($1,$2,$3) RETURNING id,name`, p.TenantID, name, Normalize(name)).Scan(&b.ID, &b.Name)
	return b, classify(err)
}

func (s *PostgresRepository) Brands(ctx context.Context, p identity.Principal, after string, limit int) ([]Brand, error) {
	rows, err := db.ForTenant(s.Pool, p.TenantID).Query(ctx, `SELECT id,name FROM brands WHERE tenant_id=$1 AND ($2::uuid IS NULL OR id>$2::uuid) ORDER BY id LIMIT $3`, p.TenantID, db.NullID(after), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Brand{}
	for rows.Next() {
		var b Brand
		if err = rows.Scan(&b.ID, &b.Name); err != nil {
			return nil, err
		}
		items = append(items, b)
	}
	return items, rows.Err()
}

func (s *PostgresRepository) CreateProduct(ctx context.Context, p identity.Principal, in ProductInput) (Product, error) {
	var product Product
	if err := in.Validate(); err != nil {
		return product, err
	}
	product.ProductInput = in
	err := db.ForTenant(s.Pool, p.TenantID).QueryRow(ctx, `INSERT INTO products (tenant_id,brand_id,sku,sku_key,manufacturer_code,code_key,description) VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id,unit`, p.TenantID, in.BrandID, in.SKU, Normalize(in.SKU), in.ManufacturerCode, Normalize(in.ManufacturerCode), in.Description).Scan(&product.ID, &product.Unit)
	if err != nil {
		return product, classify(err)
	}
	return s.Product(ctx, p, product.ID)
}

func (s *PostgresRepository) Product(ctx context.Context, p identity.Principal, id string) (Product, error) {
	var v Product
	if !validate.ValidID(id) {
		return v, fault.ErrInvalid
	}
	err := db.ForTenant(s.Pool, p.TenantID).QueryRow(ctx, `SELECT p.id,p.brand_id,p.sku,p.manufacturer_code,p.description,p.unit,b.name FROM products p JOIN brands b ON b.tenant_id=p.tenant_id AND b.id=p.brand_id WHERE p.tenant_id=$1 AND p.id=$2`, p.TenantID, id).Scan(&v.ID, &v.BrandID, &v.SKU, &v.ManufacturerCode, &v.Description, &v.Unit, &v.Brand)
	return v, classify(err)
}

func (s *PostgresRepository) Products(ctx context.Context, p identity.Principal, q, after string, limit int) ([]Product, error) {
	// Escape LIKE metacharacters: searching for '%' means a literal percent.
	raw := strings.ToLower(strings.TrimSpace(q))
	q = strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(raw)
	rows, err := db.ForTenant(s.Pool, p.TenantID).Query(ctx, `SELECT p.id,p.brand_id,p.sku,p.manufacturer_code,p.description,p.unit,b.name FROM products p JOIN brands b ON b.tenant_id=p.tenant_id AND b.id=p.brand_id WHERE p.tenant_id=$1 AND ($2='' OR p.search_text LIKE '%' || $2 || '%' OR (catalog_search_key($5)<>'' AND p.search_normalized LIKE '%' || catalog_search_key($5) || '%')) AND ($3::uuid IS NULL OR p.id>$3::uuid) ORDER BY p.id LIMIT $4`, p.TenantID, q, db.NullID(after), limit, raw)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Product{}
	for rows.Next() {
		var v Product
		if err = rows.Scan(&v.ID, &v.BrandID, &v.SKU, &v.ManufacturerCode, &v.Description, &v.Unit, &v.Brand); err != nil {
			return nil, err
		}
		items = append(items, v)
	}
	return items, rows.Err()
}
