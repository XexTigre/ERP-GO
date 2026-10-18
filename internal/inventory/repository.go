package inventory

import (
	"context"
	"github.com/XexTigre/ERP-GO/internal/identity"
)

type Repository interface {
	Move(ctx context.Context, p identity.Principal, key string, in MovementInput) (Movement, bool, error)
	Movement(ctx context.Context, p identity.Principal, id string) (Movement, error)
	Balances(ctx context.Context, p identity.Principal, branch, after string, limit int) ([]Balance, error)
	Movements(ctx context.Context, p identity.Principal, f MovementFilter) ([]Movement, error)
	ProductBalances(ctx context.Context, p identity.Principal, id, after string, limit int) ([]ProductBalance, error)
}
