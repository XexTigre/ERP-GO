package catalog

import (
	"context"
	"github.com/XexTigre/ERP-GO/internal/identity"
)

type Repository interface {
	CreateBrand(ctx context.Context, p identity.Principal, name string) (Brand, error)
	Brands(ctx context.Context, p identity.Principal, after string, limit int) ([]Brand, error)
	CreateProduct(ctx context.Context, p identity.Principal, in ProductInput) (Product, error)
	Product(ctx context.Context, p identity.Principal, id string) (Product, error)
	Products(ctx context.Context, p identity.Principal, q, after string, limit int) ([]Product, error)
}
