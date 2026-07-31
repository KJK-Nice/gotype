package player

import (
	"crypto/rand"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// Crockford base32 alphabet (no I, L, O, U).
const crockford = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

const claimRawLen = 12

// GenerateClaimCode returns a 12-char Crockford Claim Code (unformatted).
func GenerateClaimCode() (string, error) {
	var out [claimRawLen]byte
	buf := make([]byte, claimRawLen)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	for i := 0; i < claimRawLen; i++ {
		out[i] = crockford[int(buf[i])%len(crockford)]
	}
	return string(out[:]), nil
}

// FormatClaimCode renders XXXX-XXXX-XXXX.
func FormatClaimCode(raw string) string {
	n := NormalizeClaimCode(raw)
	if len(n) != claimRawLen {
		return n
	}
	return n[:4] + "-" + n[4:8] + "-" + n[8:]
}

// NormalizeClaimCode uppercases and strips separators / ambiguous glyphs.
func NormalizeClaimCode(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, " ", "")
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case 'I', 'L':
			b.WriteByte('1')
		case 'O':
			b.WriteByte('0')
		case 'U':
			// U is not in Crockford; reject later via ValidClaimCode
			b.WriteRune(r)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// ValidClaimCode reports whether raw (normalized) is 12 Crockford chars.
func ValidClaimCode(raw string) bool {
	n := NormalizeClaimCode(raw)
	if len(n) != claimRawLen {
		return false
	}
	for _, r := range n {
		if !strings.ContainsRune(crockford, r) {
			return false
		}
	}
	return true
}

// ParseClaimCode normalizes display or raw input.
func ParseClaimCode(s string) (string, error) {
	n := NormalizeClaimCode(s)
	if !ValidClaimCode(n) {
		return "", fmt.Errorf("invalid claim code")
	}
	return n, nil
}

// HashClaimCode stores only a password hash of the Claim Code.
func HashClaimCode(raw string) (string, error) {
	n, err := ParseClaimCode(raw)
	if err != nil {
		return "", err
	}
	h, err := bcrypt.GenerateFromPassword([]byte(n), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(h), nil
}

// VerifyClaimCode checks plaintext against a stored hash.
func VerifyClaimCode(raw, hash string) bool {
	n, err := ParseClaimCode(raw)
	if err != nil {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(n)) == nil
}
