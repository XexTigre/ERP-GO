package inventory

import "testing"

func TestMovementRejectsAmbiguousQuantities(t *testing.T) {
	for _, quantity := range []int64{-1, 0, 1_000_000_001} {
		in := MovementInput{BranchID: "11111111-1111-1111-1111-111111111111", ProductID: "22222222-2222-2222-2222-222222222222", Kind: "entrada", Quantity: quantity, Reason: "Contagem"}
		if in.Validate("request_1234567890") == nil {
			t.Fatalf("aceitou quantidade %d", quantity)
		}
	}
}
