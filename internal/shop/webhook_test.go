package shop

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"
)

func TestVerifyPhoenixdSignature(t *testing.T) {
	body := []byte(`{"type":"payment_received","timestamp":1712785550079,"amountSat":8,"paymentHash":"abc","externalId":null}`)
	secret := "test-webhook-secret"
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	want := hex.EncodeToString(mac.Sum(nil))

	if err := VerifyPhoenixdSignature(body, want, secret); err != nil {
		t.Fatal(err)
	}
	if err := VerifyPhoenixdSignature(body, "deadbeef", secret); err == nil {
		t.Fatal("expected signature failure")
	}
	if err := VerifyPhoenixdSignature(body, "", secret); err == nil {
		t.Fatal("expected missing signature")
	}
	// Empty secret skips verify.
	if err := VerifyPhoenixdSignature(body, "", ""); err != nil {
		t.Fatal(err)
	}
}

func TestParsePhoenixdWebhookSkew(t *testing.T) {
	now := time.UnixMilli(1712785550079).UTC()
	body := []byte(`{"type":"payment_received","timestamp":1712785550079,"amountSat":21,"paymentHash":"abc","externalId":"tip_1"}`)
	ev, err := ParsePhoenixdWebhook(body, now, DefaultWebhookSkew)
	if err != nil {
		t.Fatal(err)
	}
	if ev.ExternalID != "tip_1" || ev.AmountSat != 21 {
		t.Fatalf("%+v", ev)
	}
	stale := now.Add(10 * time.Minute)
	if _, err := ParsePhoenixdWebhook(body, stale, DefaultWebhookSkew); err == nil {
		t.Fatal("expected skew rejection")
	}
}

func TestClassifyExternalID(t *testing.T) {
	if ClassifyExternalID("ord_abc") != ExternalOrder {
		t.Fatal("order")
	}
	if ClassifyExternalID("tip_abc") != ExternalTip {
		t.Fatal("tip")
	}
	if ClassifyExternalID("x") != ExternalUnknown {
		t.Fatal("unknown")
	}
}
