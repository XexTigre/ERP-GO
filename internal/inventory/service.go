package inventory

import (
	"context"
	"github.com/XexTigre/ERP-GO/internal/identity"
	"github.com/XexTigre/ERP-GO/internal/platform/fault"
)

type Service struct{ repo Repository }

func New(repo Repository) *Service { return &Service{repo: repo} }
func (s *Service) Move(ctx context.Context, p identity.Principal, key string, in MovementInput) (Movement, bool, error) {
	if !p.CanWrite(identity.ResourceStock) {
		return Movement{}, false, fault.ErrForbidden
	}
	if err := in.Validate(key); err != nil {
		return Movement{}, false, err
	}
	return s.repo.Move(ctx, p, key, in)
}
func (s *Service) Movement(ctx context.Context, p identity.Principal, id string) (Movement, error) {
	return s.repo.Movement(ctx, p, id)
}
func (s *Service) Balances(ctx context.Context, p identity.Principal, branch, after string, limit int) ([]Balance, error) {
	return s.repo.Balances(ctx, p, branch, after, limit)
}
func (s *Service) Movements(ctx context.Context, p identity.Principal, f MovementFilter) ([]Movement, error) {
	if err := f.Validate(); err != nil {
		return nil, err
	}
	return s.repo.Movements(ctx, p, f)
}
func (s *Service) ProductBalances(ctx context.Context, p identity.Principal, id, after string, limit int) ([]ProductBalance, error) {
	return s.repo.ProductBalances(ctx, p, id, after, limit)
}
