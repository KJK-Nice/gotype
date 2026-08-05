package lnauth

import (
	"encoding/hex"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/ecdsa"
)

func TestNormalizeLinkingKey(t *testing.T) {
	priv, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	compressed := hex.EncodeToString(priv.PubKey().SerializeCompressed())
	got, err := NormalizeLinkingKey(compressed)
	if err != nil {
		t.Fatal(err)
	}
	if got != compressed {
		t.Fatalf("got %q want %q", got, compressed)
	}
}

func TestVerifyRoundTrip(t *testing.T) {
	k1, err := GenerateK1()
	if err != nil {
		t.Fatal(err)
	}
	priv, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	msg, err := hex.DecodeString(k1)
	if err != nil {
		t.Fatal(err)
	}
	sig := ecdsa.Sign(priv, msg)
	sigHex := hex.EncodeToString(sig.Serialize())
	keyHex := hex.EncodeToString(priv.PubKey().SerializeCompressed())
	ok, err := Verify(k1, sigHex, keyHex)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected valid signature")
	}
}

func TestAuthURLAndEncode(t *testing.T) {
	k1 := "a1b2c3d4e5f6789012345678901234567890abcdef1234567890abcdef123456"
	u, err := AuthURL("https://gotype.fun", k1, ActionLogin, "sess-1")
	if err != nil {
		t.Fatal(err)
	}
	if u == "" {
		t.Fatal("empty url")
	}
	ln, err := EncodeLNURL(u)
	if err != nil {
		t.Fatal(err)
	}
	if ln[:5] != "LNURL" {
		t.Fatalf("want LNURL prefix, got %q", ln[:10])
	}
}
