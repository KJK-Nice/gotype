package player

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/btcsuite/btcd/btcec/v2"
)

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