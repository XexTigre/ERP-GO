package identity

import (
	"context"
	"github.com/XexTigre/ERP-GO/internal/platform/fault"
	"github.com/XexTigre/ERP-GO/internal/platform/validate"
)

type Service struct{ repo Repository }

func New(repo Repository) *Service { return &Service{repo: repo} }
func (s *Service) Authenticate(ctx context.Context, raw string) (Principal, error) {
	return s.repo.Authenticate(ctx, raw)
}
func (s *Service) CreateBranch(ctx context.Context, p Principal, name string) (Branch, error) {
	if !p.CanWrite(ResourceAdmin) {
		return Branch{}, fault.ErrForbidden
	}
	name, err := validate.CleanText(name, 120)
	if err != nil {
		return Branch{}, err
	}
	return s.repo.CreateBranch(ctx, p, name)
}
func (s *Service) Branches(ctx context.Context, p Principal, after string, limit int) ([]Branch, error) {
	return s.repo.Branches(ctx, p, after, limit)
}
