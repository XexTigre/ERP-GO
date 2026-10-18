package httpx

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecoderRejectsUnexpectedJSON(t *testing.T) {
	for _, body := range []string{`{"name":"Teste","tenant_id":"outro"}`, `{"name":"Teste"} {}`, `{`, `{"name":"` + strings.Repeat("x", 65536) + `"}`} {
		r := httptest.NewRequest("POST", "/", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		var in struct {
			Name string `json:"name"`
		}
		if Decode(w, r, &in) {
			t.Fatal("JSON ambiguo aceito")
		}
	}
}
