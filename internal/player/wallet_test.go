package player

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/kjkusap/monkeytype-clone/internal/persist"
)

func TestRegisterAndLoginWithLinkingKey(t *testing.T) {
	dir := t.TempDir()
	store, err := persist.Open(filepath.Join(dir, "gotype.json"))
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(store)
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	key := "02c3b844b8104f0c1b15c507774c9ba7fc609f58f343b9b149122e944dd20c9362"

	p, err := svc.RegisterWithLinkingKey("WalletNeo", key, "1.2.3.4", "sess-a", now)
	if err != nil {
		t.Fatal(err)
	}
	if p.LinkingKey == "" {
		t.Fatal("expected linking key stored")
	}

	got, err := svc.LoginWithLinkingKey(key, "9.9.9.9", "sess-b", now)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != p.ID || got.ActiveSessionID != "sess-b" {
		t.Fatalf("unexpected login %+v", got)
	}
}

func TestLinkWallet(t *testing.T) {
	dir := t.TempDir()
	store, err := persist.Open(filepath.Join(dir, "gotype.json"))
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(store)
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	reg, err := svc.Register("Claimer", "1.1.1.1", "sess-a", now)
	if err != nil {
		t.Fatal(err)
	}
	key := "03aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if err := svc.LinkWallet(reg.Player.ID, key); err != nil {
		t.Fatal(err)
	}
	p, err := store.GetPlayer(reg.Player.ID)
	if err != nil {
		t.Fatal(err)
	}
	if p.LinkingKey == "" {
		t.Fatal("linking key not saved")
	}
}
