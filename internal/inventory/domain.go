package inventory

import (
	"fmt"
	"github.com/XexTigre/ERP-GO/internal/platform/fault"
	"github.com/XexTigre/ERP-GO/internal/platform/validate"
	"regexp"
	"strings"
	"time"
)

var (
	ErrInsufficient        = &fault.Coded{Kind: fault.ErrConflict, Code: "insufficient_stock", Message: "Saldo insuficiente ou limite de saldo excedido"}
	ErrIdempotencyConflict = &fault.Coded{Kind: fault.ErrConflict, Code: "idempotency_conflict", Message: "Chave de idempotência reutilizada com outros dados"}
)
var keyPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{16,100}$`)

type MovementInput struct {
	BranchID  string `json:"branch_id"`
	ProductID string `json:"product_id"`
	Kind      string `json:"kind"`
	Quantity  int64  `json:"quantity"`
	Reason    string `json:"reason"`
}

type Movement struct {
	ID string `json:"id"`
	MovementInput
	IdempotencyKey string    `json:"idempotency_key"`
	CreatedBy      string    `json:"created_by"`
	CreatedAt      time.Time `json:"created_at"`
	BalanceAfter   int64     `json:"balance_after"`
}

type Balance struct {
	ProductID   string `json:"product_id"`
	SKU         string `json:"sku"`
	Description string `json:"description"`
	Quantity    int64  `json:"quantity"`
}

type ProductBalance struct {
	BranchID string `json:"branch_id"`
	Branch   string `json:"branch"`
	Quantity int64  `json:"quantity"`
}

func (in *MovementInput) Validate(key string) error {
	if !validate.ValidID(in.BranchID) || !validate.ValidID(in.ProductID) || !keyPattern.MatchString(key) {
		return fmt.Errorf("%w: ids ou Idempotency-Key", fault.ErrInvalid)
	}
	in.BranchID = strings.ToLower(in.BranchID)
	in.ProductID = strings.ToLower(in.ProductID)
	if in.Kind != "entrada" && in.Kind != "saida" || in.Quantity < 1 || in.Quantity > 1_000_000_000 {
		return fmt.Errorf("%w: kind ou quantity (inteiro positivo)", fault.ErrInvalid)
	}
	var err error
	in.Reason, err = validate.CleanText(in.Reason, 300)
	return err
}

func (in MovementInput) Delta() int64 {
	if in.Kind == "saida" {
		return -in.Quantity
	}
	return in.Quantity
}
