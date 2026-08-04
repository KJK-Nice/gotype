package ln

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/kjkusap/monkeytype-clone/internal/persist"
	"github.com/kjkusap/monkeytype-clone/internal/shop"
)

const tipInvoiceTTL = 15 * time.Minute

// Invoice is a payable bolt11 tip.
type Invoice struct {
	ID          string // tip_<id> when tracked (Phoenixd); empty for LNURL-only
	Bolt11      string
	Sats        int
	PaymentHash string
	Tracked     bool // true when a TipIntent was persisted
}

// CreateInvoice returns a bolt11 tip invoice via Phoenixd (preferred) or LNURL-pay.
// When store is non-nil and Phoenixd is configured, a TipIntent is persisted and
// correlated via externalId + optional webhookUrl for settle push.
func CreateInvoice(ctx context.Context, store *persist.Store, sats int, comment string, now time.Time) (Invoice, error) {
	if sats <= 0 {
		return Invoice{}, fmt.Errorf("sats must be positive")
	}
	if cfg := shop.PhoenixdFromEnv(); cfg.Configured() {
		return createTrackedTip(ctx, store, cfg, sats, comment, now)
	}
	dest := tipDestination()
	if dest == "" {
		return Invoice{}, fmt.Errorf("set PHOENIXD_URL + PHOENIXD_PASSWORD or TIP_LIGHTNING_ADDRESS or TIP_LNURL")
	}

	payURL, err := resolvePayURL(dest)
	if err != nil {
		return Invoice{}, err
	}

	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()

	params, err := fetchPayParams(ctx, payURL)
	if err != nil {
		return Invoice{}, err
	}

	msat := int64(sats) * 1000
	if params.MinSendable > 0 && msat < params.MinSendable {
		return Invoice{}, fmt.Errorf("min tip is %d sats", params.MinSendable/1000)
	}
	if params.MaxSendable > 0 && msat > params.MaxSendable {
		return Invoice{}, fmt.Errorf("max tip is %d sats", params.MaxSendable/1000)
	}

	pr, err := fetchInvoice(ctx, params.Callback, msat, comment, params.CommentAllowed)
	if err != nil {
		return Invoice{}, err
	}
	return Invoice{Bolt11: pr, Sats: sats}, nil
}

func createTrackedTip(ctx context.Context, store *persist.Store, cfg shop.PhoenixdConfig, sats int, comment string, now time.Time) (Invoice, error) {
	if store == nil {
		return Invoice{}, fmt.Errorf("tip store required for Phoenixd tips")
	}
	id, err := newTipID()
	if err != nil {
		return Invoice{}, err
	}
	tip := persist.TipIntent{
		ID:        id,
		Sats:      sats,
		State:     persist.TipPending,
		CreatedAt: now.UTC(),
	}
	if err := store.SaveTip(tip); err != nil {
		return Invoice{}, err
	}
	created, err := shop.NewPhoenixdClient(cfg).CreateInbound(ctx, sats, comment, id, int(tipInvoiceTTL.Seconds()))
	if err != nil {
		return Invoice{}, err
	}
	tip.Bolt11 = created.PaymentRequest
	tip.PaymentHash = created.PaymentHash
	tip.CheckingID = created.CheckingID
	if err := store.SaveTip(tip); err != nil {
		return Invoice{}, err
	}
	return Invoice{
		ID:          tip.ID,
		Bolt11:      tip.Bolt11,
		Sats:        tip.Sats,
		PaymentHash: tip.PaymentHash,
		Tracked:     true,
	}, nil
}

// PollTip confirms payment via Phoenixd poll and marks the TipIntent paid (idempotent).
func PollTip(ctx context.Context, store *persist.Store, tipID string, now time.Time) (persist.TipIntent, error) {
	if store == nil {
		return persist.TipIntent{}, fmt.Errorf("tip store required")
	}
	tip, err := store.GetTip(tipID)
	if err != nil {
		return persist.TipIntent{}, err
	}
	if tip.State == persist.TipPaid {
		return tip, nil
	}
	cfg := shop.PhoenixdFromEnv()
	if !cfg.Configured() {
		return tip, fmt.Errorf("phoenixd not configured")
	}
	st, err := shop.NewPhoenixdClient(cfg).CheckPaid(ctx, tip.CheckingID)
	if err != nil {
		return tip, err
	}
	if !st.Paid {
		return tip, nil
	}
	if tip.PaymentHash != "" && st.PaymentHash != "" && st.PaymentHash != tip.PaymentHash {
		return tip, fmt.Errorf("payment_hash mismatch")
	}
	return store.MarkTipPaid(tip.ID, now)
}

// SettleTipFromWebhook marks a tip paid after authoritative CheckPaid (webhook hint path).
func SettleTipFromWebhook(ctx context.Context, store *persist.Store, ev shop.PhoenixdWebhookEvent, now time.Time) (persist.TipIntent, error) {
	if store == nil {
		return persist.TipIntent{}, fmt.Errorf("tip store required")
	}
	var tip persist.TipIntent
	var err error
	if shop.ClassifyExternalID(ev.ExternalID) == shop.ExternalTip {
		tip, err = store.GetTip(ev.ExternalID)
	} else {
		tip, err = store.FindTipByPaymentHash(ev.PaymentHash)
	}
	if err != nil {
		return persist.TipIntent{}, err
	}
	return PollTip(ctx, store, tip.ID, now)
}

func newTipID() (string, error) {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return fmt.Sprintf("tip_%x", b[:]), nil
}
