package shop

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kjkusap/monkeytype-clone/internal/catalog"
	"github.com/kjkusap/monkeytype-clone/internal/persist"
	"github.com/kjkusap/monkeytype-clone/internal/player"
	"github.com/kjkusap/monkeytype-clone/internal/progress"
)

func signBody(t *testing.T, body []byte, secret string) string {
	t.Helper()
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func TestWebhookHandlerBuyGrant(t *testing.T) {
	dir := t.TempDir()
	store, err := persist.Open(filepath.Join(dir, "gotype.json"))
	if err != nil {
		t.Fatal(err)
	}
	ps := player.NewService(store)
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	reg, err := ps.Register("WebBuyer", "2.2.2.2", "s", now)
	if err != nil {
		t.Fatal(err)
	}
	prog := progress.NewService(store)
	inv := &fakeInv{paid: true}
	svc := NewService(store, inv, prog)
	o, err := svc.CreateBuy(context.Background(), reg.Player.ID, catalog.SKUHeart, now)
	if err != nil {
		t.Fatal(err)
	}

	secret := "whsec"
	h := &WebhookHandler{
		Store:  store,
		Shop:   svc,
		Secret: secret,
		Now:    func() time.Time { return now },
	}
	body, _ := json.Marshal(map[string]any{
		"type":        "payment_received",
		"timestamp":   now.UnixMilli(),
		"amountSat":   o.Sats,
		"paymentHash": o.PaymentHash,
		"externalId":  o.ID,
	})
	req := httptest.NewRequest(http.MethodPost, WebhookPath, strings.NewReader(string(body)))
	req.Header.Set(PhoenixdSignatureHeader, signBody(t, body, secret))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	got, err := store.GetOrder(o.ID)
	if err != nil || got.State != persist.OrderGranted {
		t.Fatalf("order=%+v err=%v", got, err)
	}
	if store.InventoryQty(reg.Player.ID, catalog.SKUHeart) != 1 {
		t.Fatal("expected heart granted")
	}
}

func TestWebhookHandlerTipSettle(t *testing.T) {
	dir := t.TempDir()
	store, err := persist.Open(filepath.Join(dir, "gotype.json"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	tip := persist.TipIntent{
		ID:          "tip_deadbeef",
		Sats:        21,
		State:       persist.TipPending,
		PaymentHash: "thash",
		CheckingID:  "thash",
		CreatedAt:   now,
	}
	if err := store.SaveTip(tip); err != nil {
		t.Fatal(err)
	}

	settler := TipSettlerFunc(func(_ context.Context, s *persist.Store, ev PhoenixdWebhookEvent, at time.Time) (persist.TipIntent, error) {
		if ev.ExternalID != tip.ID {
			t.Fatalf("externalId=%s", ev.ExternalID)
		}
		return s.MarkTipPaid(tip.ID, at)
	})
	secret := "whsec"
	h := &WebhookHandler{
		Store:  store,
		Tips:   settler,
		Secret: secret,
		Now:    func() time.Time { return now },
	}
	body, _ := json.Marshal(map[string]any{
		"type":        "payment_received",
		"timestamp":   now.UnixMilli(),
		"amountSat":   21,
		"paymentHash": "thash",
		"externalId":  tip.ID,
	})
	req := httptest.NewRequest(http.MethodPost, WebhookPath, strings.NewReader(string(body)))
	req.Header.Set(PhoenixdSignatureHeader, signBody(t, body, secret))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	got, err := store.GetTip(tip.ID)
	if err != nil || got.State != persist.TipPaid {
		t.Fatalf("tip=%+v err=%v", got, err)
	}
}

func TestWebhookHandlerRejectsBadSignature(t *testing.T) {
	h := &WebhookHandler{Secret: "whsec", Now: time.Now}
	body := []byte(`{"type":"payment_received","timestamp":1,"paymentHash":"x","externalId":"tip_1"}`)
	req := httptest.NewRequest(http.MethodPost, WebhookPath, strings.NewReader(string(body)))
	req.Header.Set(PhoenixdSignatureHeader, "nope")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", rr.Code)
	}
}
