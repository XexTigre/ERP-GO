package catalog

import (
	"fmt"
	"github.com/XexTigre/ERP-GO/internal/platform/fault"
	"github.com/XexTigre/ERP-GO/internal/platform/validate"
	"strings"
)

var (
	ErrDuplicateIdentity = &fault.Coded{Kind: fault.ErrConflict, Code: "duplicate_product_identity", Message: "Marca e código do fabricante já cadastrados"}
	ErrDuplicateSKU      = &fault.Coded{Kind: fault.ErrConflict, Code: "duplicate_sku", Message: "SKU interno já cadastrado"}
)

type Brand struct {
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

func Normalize(value string) string { return strings.ToUpper(strings.Join(strings.Fields(value), " ")) }

func (in *ProductInput) Validate() error {
	if !validate.ValidID(in.BrandID) {
		return fmt.Errorf("%w: brand_id", fault.ErrInvalid)
	}
	in.BrandID = strings.ToLower(in.BrandID)
	var err error
	if in.SKU, err = validate.CleanText(in.SKU, 80); err != nil {
		return fmt.Errorf("%w: sku", err)
	}
	if in.ManufacturerCode, err = validate.CleanText(in.ManufacturerCode, 120); err != nil {
		return fmt.Errorf("%w: manufacturer_code", err)
	}
	if in.Description, err = validate.CleanText(in.Description, 300); err != nil {
		return fmt.Errorf("%w: description", err)
	}
	return nil
}
