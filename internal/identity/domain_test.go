package identity

import "testing"

func TestRoles(t *testing.T) {
	for _, role := range []string{"admin", "cataloger", "stock", "reader", "unknown"} {
		p := Principal{Role: Role(role)}
		for _, resource := range []string{"admin", "catalog", "stock"} {
			want := role == "admin" || role == "cataloger" && resource == "catalog" || role == "stock" && resource == "stock"
			if p.CanWrite(Resource(resource)) != want {
				t.Fatalf("permissao inesperada: %s / %s", role, resource)
			}
		}
	}
}
