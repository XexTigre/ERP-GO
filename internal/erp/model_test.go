package erp

import "testing"

func TestPartIdentityPreservesDifferentiators(t *testing.T) {
	codes := []string{"101.24 AC", "101.24 BL", "00123", "123", "A/01", "A-01"}
	seen := map[string]bool{}
	for _, code := range codes {
		key := Normalize(code)
		if seen[key] {
			t.Fatalf("identidades diferentes foram fundidas: %q", code)
		}
		seen[key] = true
	}
	if Normalize(" 101.24   bl ") != "101.24 BL" {
		t.Fatal("espacos e caixa nao normalizados")
	}
}

func TestRoles(t *testing.T) {
	for _, role := range []string{"admin", "cataloger", "stock", "reader", "unknown"} {
		p := Principal{Role: role}
		for _, resource := range []string{"admin", "catalog", "stock"} {
			want := role == "admin" || role == "cataloger" && resource == "catalog" || role == "stock" && resource == "stock"
			if p.CanWrite(resource) != want {
				t.Fatalf("permissao inesperada: %s / %s", role, resource)
			}
		}
	}
}

func TestMovementRejectsAmbiguousQuantities(t *testing.T) {
	for _, quantity := range []int64{-1, 0, 1_000_000_001} {
		in := MovementInput{BranchID: "11111111-1111-1111-1111-111111111111", ProductID: "22222222-2222-2222-2222-222222222222", Kind: "entrada", Quantity: quantity, Reason: "Contagem"}
		if in.Validate("request_1234567890") == nil {
			t.Fatalf("aceitou quantidade %d", quantity)
		}
	}
}
