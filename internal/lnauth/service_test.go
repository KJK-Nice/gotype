package lnauth

import (
	"encoding/hex"
	"path/filepath"
	"testing"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/ecdsa"

	"github.com/kjkusap/monkeytype-clone/internal/persist"
	"github.com/kjkusap/monkeytype-clone/internal/player"
)

func TestServiceWalletRegisterFlow(t *testing.T) {
	t.Setenv("GOTYPE_PUBLIC_URL", "https://gotype.fun")
	t.Setenv("REDIS_URL", "")
	dir := t.TempDir()
	store, err := persist.Open(filepath.Join(dir, "gotype.json"))
	if err != nil {
		t.Fatal(err)
	}
	players := player.NewService(store)
	svc := NewService(players)
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)

	start, err := svc.Start("sess-1", ActionLogin, "", now)
	if err != nil {
		t.Fatal(err)
	}
	priv, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	msg, _ := hex.DecodeString(start.K1)
	sig := ecdsa.Sign(priv, msg)
	key := hex.EncodeToString(priv.PubKey().SerializeCompressed())
	if err := svc.HandleCallback(start.K1, hex.EncodeToString(sig.Serialize()), key, "1.2.3.4", now); err != nil {
		t.Fatal(err)
	}
	st, err := svc.Status(start.K1)
	if err != nil {
		t.Fatal(err)
	}
	if st.State != StateVerified {
		t.Fatalf("state = %q", st.State)
	}
	p, err := svc.CompleteRegister(start.K1, "LightUser", "1.2.3.4", now)
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "LightUser" {
		t.Fatalf("name = %q", p.Name)
	}
	st, err = svc.Status(start.K1)
	if err != nil {
		t.Fatal(err)
	}
	if st.State != StateOK || st.PlayerID != p.ID {
		t.Fatalf("final status %+v", st)
	}
}

func TestServiceReturningWalletLogin(t *testing.T) {
	t.Setenv("GOTYPE_PUBLIC_URL", "https://gotype.fun")
	t.Setenv("REDIS_URL", "")
	dir := t.TempDir()
	store, err := persist.Open(filepath.Join(dir, "gotype.json"))
	if err != nil {
		t.Fatal(err)
	}
	players := player.NewService(store)
	svc := NewService(players)
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)

	priv, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	key := hex.EncodeToString(priv.PubKey().SerializeCompressed())
	if _, err := players.RegisterWithLinkingKey("OldWallet", key, "1.1.1.1", "sess-old", now); err != nil {
		t.Fatal(err)
	}

	start, err := svc.Start("sess-new", ActionLogin, "", now)
	if err != nil {
		t.Fatal(err)
	}
	msg, _ := hex.DecodeString(start.K1)
	sig := ecdsa.Sign(priv, msg)
	if err := svc.HandleCallback(start.K1, hex.EncodeToString(sig.Serialize()), key, "2.2.2.2", now); err != nil {
		t.Fatal(err)
	}
	st, err := svc.Status(start.K1)
	if err != nil {
		t.Fatal(err)
	}
	if st.State != StateOK || st.Name != "OldWallet" {
		t.Fatalf("status %+v", st)
	}
}
