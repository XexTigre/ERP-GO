package erp

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	ErrInvalid      = errors.New("dados invalidos")
	ErrNotFound     = errors.New("registro nao encontrado")
	ErrConflict     = errors.New("registro em conflito")
	ErrInsufficient = errors.New("saldo insuficiente")
)

type Principal struct {
	TenantID string
	KeyID    string
	Role     string
}

func (p Principal) CanWrite(resource string) bool {
	return p.Role == "admin" || p.Role == "cataloger" && resource == "catalog" || p.Role == "stock" && resource == "stock"
}

type Brand struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Branch struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ProductInput struct {
	BrandID          string `json:"brand_id"`
	SKU              string `json:"sku"`
	ManufacturerCode string `json:"manufacturer_code"`
	Description      string `json:"description"`
}

type Product struct {
	ID string `json:"id"`
	ProductInput
	Brand string `json:"brand"`
	Unit  string `json:"unit"`
}

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

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
var keyPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{16,100}$`)

func ValidID(id string) bool { return uuidPattern.MatchString(id) }

// Normalize preserves dots, slashes, suffixes and leading zeroes in part codes.
func Normalize(value string) string { return strings.ToUpper(strings.Join(strings.Fields(value), " ")) }

func CleanText(value string, max int) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || !utf8.ValidString(value) || utf8.RuneCountInString(value) > max || strings.ContainsAny(value, "\x00\r\n") {
		return "", ErrInvalid
	}
	return value, nil
}

func (in *ProductInput) Validate() error {
	if !ValidID(in.BrandID) {
		return fmt.Errorf("%w: brand_id", ErrInvalid)
	}
	in.BrandID = strings.ToLower(in.BrandID)
	var err error
	if in.SKU, err = CleanText(in.SKU, 80); err != nil {
		return fmt.Errorf("%w: sku", err)
	}
	if in.ManufacturerCode, err = CleanText(in.ManufacturerCode, 120); err != nil {
		return fmt.Errorf("%w: manufacturer_code", err)
	}
	if in.Description, err = CleanText(in.Description, 300); err != nil {
		return fmt.Errorf("%w: description", err)
	}
	return nil
}

func (in *MovementInput) Validate(key string) error {
	if !ValidID(in.BranchID) || !ValidID(in.ProductID) || !keyPattern.MatchString(key) {
		return fmt.Errorf("%w: ids ou Idempotency-Key", ErrInvalid)
	}
	in.BranchID = strings.ToLower(in.BranchID)
	in.ProductID = strings.ToLower(in.ProductID)
	if in.Kind != "entrada" && in.Kind != "saida" || in.Quantity < 1 || in.Quantity > 1_000_000_000 {
		return fmt.Errorf("%w: kind ou quantity (inteiro positivo)", ErrInvalid)
	}
	var err error
	in.Reason, err = CleanText(in.Reason, 300)
	return err
}

func (in MovementInput) Delta() int64 {
	if in.Kind == "saida" {
		return -in.Quantity
	}
	return in.Quantity
}
