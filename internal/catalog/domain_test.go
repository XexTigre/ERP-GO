package catalog

import "testing"

func FuzzNormalizeIdentity(f *testing.F) {
	for _, seed := range []string{"001.24 BL", "001.24 AC", "A/01", "A-01", "  código   completo  "} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, s string) {
		normalized := Normalize(s)
		if Normalize(normalized) != normalized {
			t.Fatalf("normalização não idempotente: %q", s)
		}
	})
}

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
