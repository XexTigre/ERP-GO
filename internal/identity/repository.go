package identity

import (
	"context"
)

type Repository interface {
	Authenticate(ctx context.Context, raw string) (Principal, error)
	CreateBranch(ctx context.Context, p Principal, name string) (Branch, error)
	Branches(ctx context.Context, p Principal, after string, limit int) ([]Branch, error)
}
