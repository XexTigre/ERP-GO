package inventory

import (
	"testing"
	"time"
)

func TestHistoryCursorKeepsTimestampAndID(t *testing.T) {
	m := Movement{ID: "11111111-1111-1111-1111-111111111111", CreatedAt: time.Date(2026, 10, 18, 12, 0, 0, 123456000, time.UTC)}
	c, err := parseCursor(MovementCursor(m))
	if err != nil || c.id != m.ID || !c.at.Equal(m.CreatedAt) {
		t.Fatalf("cursor: %+v %v", c, err)
	}
	for _, raw := range []string{"invalid", "aW52YWxpZA", string(make([]byte, 181))} {
		if _, err = parseCursor(raw); err == nil {
			t.Fatal("cursor inválido aceito")
		}
	}
}
