package player

import (
	"fmt"
	"strings"
	"unicode"
)

const (
	NameMin = 3
	NameMax = 16
)

// NormalizeName validates display name: 3–16 [a-zA-Z0-9_].
func NormalizeName(s string) (string, error) {
	s = strings.TrimSpace(s)
	if len(s) < NameMin || len(s) > NameMax {
		return "", fmt.Errorf("name must be %d–%d characters", NameMin, NameMax)
	}
	for _, r := range s {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_') {
			return "", fmt.Errorf("name may only use letters, digits, underscore")
		}
		if r > unicode.MaxASCII {
			return "", fmt.Errorf("name may only use letters, digits, underscore")
		}
	}
	return s, nil
}

// NameKey is the case-insensitive uniqueness key.
func NameKey(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
