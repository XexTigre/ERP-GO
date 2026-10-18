package validate

import (
	"github.com/XexTigre/ERP-GO/internal/platform/fault"
	"regexp"
	"strings"
	"unicode/utf8"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func ValidID(id string) bool { return uuidPattern.MatchString(id) }

// Normalize preserves dots, slashes, suffixes and leading zeroes in part codes.

func CleanText(value string, max int) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || !utf8.ValidString(value) || utf8.RuneCountInString(value) > max || strings.ContainsAny(value, "\x00\r\n") {
		return "", fault.ErrInvalid
	}
	return value, nil
}
