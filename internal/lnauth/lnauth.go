package lnauth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"

	"github.com/btcsuite/btcd/btcec/v2"
	lnurl "github.com/fiatjaf/go-lnurl"
)

// Action is the LUD-04 action query parameter.
type Action string

const (
	ActionRegister Action = "register"
	ActionLogin    Action = "login"
	ActionLink     Action = "link"
)

// GenerateK1 returns a random 32-byte hex challenge.
func GenerateK1() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

// AuthURL builds the LNURL-auth callback URL embedded in the QR.
func AuthURL(base, k1 string, action Action, sid string) (string, error) {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		return "", fmt.Errorf("public base URL required")
	}
	if len(k1) != 64 {
		return "", fmt.Errorf("k1 must be 64 hex chars")
	}
	u, err := url.Parse(base + "/auth/lnurl")
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("tag", "login")
	q.Set("k1", k1)
	q.Set("action", string(action))
	if sid != "" {
		q.Set("sid", sid)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// EncodeLNURL bech32-encodes a callback URL (LUD-01).
func EncodeLNURL(callback string) (string, error) {
	return lnurl.LNURLEncode(callback)
}

// Verify checks a LUD-04 wallet signature.
func Verify(k1, sig, key string) (bool, error) {
	return lnurl.VerifySignature(k1, sig, key)
}

// NormalizeLinkingKey returns compressed 33-byte secp256k1 pubkey hex.
func NormalizeLinkingKey(key string) (string, error) {
	key = strings.TrimSpace(key)
	b, err := hex.DecodeString(key)
	if err != nil {
		return "", fmt.Errorf("invalid linking key hex")
	}
	pub, err := btcec.ParsePubKey(b)
	if err != nil {
		return "", fmt.Errorf("invalid linking key: %w", err)
	}
	return hex.EncodeToString(pub.SerializeCompressed()), nil
}
