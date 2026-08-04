package shop

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	// PhoenixdSignatureHeader is the HMAC-SHA256 of the raw JSON body.
	PhoenixdSignatureHeader = "X-Phoenix-Signature"
	// DefaultWebhookSkew is the max age for payment_received timestamps (Stripe-style).
	DefaultWebhookSkew = 5 * time.Minute
)

// PhoenixdWebhookEvent is the payment_received body Phoenixd POSTs to webhookUrl.
type PhoenixdWebhookEvent struct {
	Type        string `json:"type"`
	Timestamp   int64  `json:"timestamp"` // unix millis
	AmountSat   int    `json:"amountSat"`
	PaymentHash string `json:"paymentHash"`
	ExternalID  string `json:"externalId"`
}

// WebhookSecretFromEnv reads PHOENIXD_WEBHOOK_SECRET (must match phoenixd --webhook-secret).
func WebhookSecretFromEnv() string {
	return strings.TrimSpace(os.Getenv("PHOENIXD_WEBHOOK_SECRET"))
}

// VerifyPhoenixdSignature checks X-Phoenix-Signature = HMAC-SHA256(body, secret) hex.
// When secret is empty, verification is skipped (local/dev only).
func VerifyPhoenixdSignature(body []byte, signatureHeader, secret string) error {
	if secret == "" {
		return nil
	}
	sig := strings.TrimSpace(signatureHeader)
	if sig == "" {
		return fmt.Errorf("missing %s", PhoenixdSignatureHeader)
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	want := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(strings.ToLower(sig)), []byte(strings.ToLower(want))) {
		return fmt.Errorf("invalid phoenixd webhook signature")
	}
	return nil
}

// ParsePhoenixdWebhook unmarshals and validates a payment_received event.
func ParsePhoenixdWebhook(body []byte, now time.Time, skew time.Duration) (PhoenixdWebhookEvent, error) {
	var ev PhoenixdWebhookEvent
	if err := json.Unmarshal(body, &ev); err != nil {
		return ev, err
	}
	if ev.Type != "" && ev.Type != "payment_received" {
		return ev, fmt.Errorf("unexpected webhook type %q", ev.Type)
	}
	if ev.PaymentHash == "" {
		return ev, fmt.Errorf("missing paymentHash")
	}
	if skew <= 0 {
		skew = DefaultWebhookSkew
	}
	if ev.Timestamp > 0 {
		ts := time.UnixMilli(ev.Timestamp).UTC()
		if ts.After(now.UTC().Add(skew)) || now.UTC().Sub(ts) > skew {
			return ev, fmt.Errorf("webhook timestamp outside skew window")
		}
	}
	return ev, nil
}

// ExternalKind classifies Phoenixd externalId prefixes.
type ExternalKind int

const (
	ExternalUnknown ExternalKind = iota
	ExternalOrder
	ExternalTip
)

// ClassifyExternalID returns Order vs Tip from externalId prefix.
func ClassifyExternalID(id string) ExternalKind {
	switch {
	case strings.HasPrefix(id, "ord_"):
		return ExternalOrder
	case strings.HasPrefix(id, "tip_"):
		return ExternalTip
	default:
		return ExternalUnknown
	}
}
