package catalog

import (
	"context"
	"github.com/XexTigre/ERP-GO/internal/identity"
	"github.com/XexTigre/ERP-GO/internal/platform/fault"
	"github.com/XexTigre/ERP-GO/internal/platform/validate"
)

type Service struct{ repo Repository }

func New(repo Repository) *Service { return &Service{repo: repo} }
func (s *Service) CreateBrand(ctx context.Context, p identity.Principal, name string) (Brand, error) {
	if !p.CanWrite(identity.ResourceCatalog) {
		return Brand{}, fault.ErrForbidden
	}
	name, err := validate.CleanText(name, 120)
	if err != nil {
		return Brand{}, err
	}
	return s.repo.CreateBrand(ctx, p, name)
}
func (s *Service) Brands(ctx context.Context, p identity.Principal, after string, limit int) ([]Brand, error) {
	return s.repo.Brands(ctx, p, after, limit)
}
func (s *Service) CreateProduct(ctx context.Context, p identity.Principal, in ProductInput) (Product, error) {
	if !p.CanWrite(identity.ResourceCatalog) {
		return Product{}, fault.ErrForbidden
	}
	if err := in.Validate(); err != nil {
		return Product{}, err
	}
	return s.repo.CreateProduct(ctx, p, in)
}
func (s *Service) Product(ctx context.Context, p identity.Principal, id string) (Product, error) {
	return s.repo.Product(ctx, p, id)
}
func (s *Service) Products(ctx context.Context, p identity.Principal, q, after string, limit int) ([]Product, error) {
	return s.repo.Products(ctx, p, q, after, limit)
}
