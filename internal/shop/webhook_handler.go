package shop

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kjkusap/monkeytype-clone/internal/persist"
)

// TipSettler settles Tip intents (implemented by ln.SettleTipFromWebhook / PollTip).
type TipSettler interface {
	SettleTip(ctx context.Context, store *persist.Store, ev PhoenixdWebhookEvent, now time.Time) (persist.TipIntent, error)
}

// TipSettlerFunc adapts a function to TipSettler.
type TipSettlerFunc func(ctx context.Context, store *persist.Store, ev PhoenixdWebhookEvent, now time.Time) (persist.TipIntent, error)

func (f TipSettlerFunc) SettleTip(ctx context.Context, store *persist.Store, ev PhoenixdWebhookEvent, now time.Time) (persist.TipIntent, error) {
	return f(ctx, store, ev, now)
}

// WebhookHandler handles Phoenixd payment_received POSTs for Buy Orders and Tips.
type WebhookHandler struct {
	Store      *persist.Store
	Shop       *Service
	Tips       TipSettler
	Secret     string
	Now        func() time.Time
	Skew       time.Duration
}

// ServeHTTP verifies signature, confirms paid via poll, then grants Order / marks Tip.
func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "read body", http.StatusBadRequest)
		return
	}
	if err := VerifyPhoenixdSignature(body, r.Header.Get(PhoenixdSignatureHeader), h.Secret); err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	now := time.Now().UTC()
	if h.Now != nil {
		now = h.Now().UTC()
	}
	skew := h.Skew
	if skew <= 0 {
		skew = DefaultWebhookSkew
	}
	ev, err := ParsePhoenixdWebhook(body, now, skew)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	switch ClassifyExternalID(ev.ExternalID) {
	case ExternalOrder:
		if h.Shop == nil {
			http.Error(w, "shop not configured", http.StatusServiceUnavailable)
			return
		}
		if _, err := h.Shop.PollAndGrant(ctx, ev.ExternalID, now); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
	case ExternalTip:
		if h.Tips == nil || h.Store == nil {
			http.Error(w, "tips not configured", http.StatusServiceUnavailable)
			return
		}
		if _, err := h.Tips.SettleTip(ctx, h.Store, ev, now); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
	default:
		// externalId missing/unknown — try tip by payment hash, then order by hash.
		if h.Store != nil && h.Tips != nil {
			if tip, err := h.Store.FindTipByPaymentHash(ev.PaymentHash); err == nil {
				if _, err := h.Tips.SettleTip(ctx, h.Store, PhoenixdWebhookEvent{
					Type:        ev.Type,
					Timestamp:   ev.Timestamp,
					AmountSat:   ev.AmountSat,
					PaymentHash: tip.PaymentHash,
					ExternalID:  tip.ID,
				}, now); err != nil {
					http.Error(w, err.Error(), http.StatusBadGateway)
					return
				}
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("ok"))
				return
			}
		}
		if h.Shop != nil && h.Store != nil {
			if o, err := findOrderByPaymentHash(h.Store, ev.PaymentHash); err == nil {
				if _, err := h.Shop.PollAndGrant(ctx, o.ID, now); err != nil {
					http.Error(w, err.Error(), http.StatusBadGateway)
					return
				}
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("ok"))
				return
			}
		}
		http.Error(w, fmt.Sprintf("unknown externalId %q", ev.ExternalID), http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func findOrderByPaymentHash(store *persist.Store, hash string) (persist.Order, error) {
	return store.FindOrderByPaymentHash(hash)
}

// WebhookPath is the HTTP path registered on gotype-ssh.
const WebhookPath = "/ln/webhook"
