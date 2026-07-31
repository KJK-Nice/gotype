package shop

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/kjkusap/monkeytype-clone/internal/catalog"
	"github.com/kjkusap/monkeytype-clone/internal/persist"
	"github.com/kjkusap/monkeytype-clone/internal/progress"
)

const InvoiceTTL = 15 * time.Minute

// Invoicer abstracts LNBits (or fakes in tests).
type Invoicer interface {
	CreateInbound(ctx context.Context, sats int, memo, orderID string, expirySec int) (CreatedInvoice, error)
	CheckPaid(ctx context.Context, checkingID string) (PaymentStatus, error)
}

// Service creates Buy Orders and grants Inventory after paid poll.
type Service struct {
	Store   *persist.Store
	Invoices Invoicer
	Progress *progress.Service
}

func NewService(store *persist.Store, inv Invoicer, prog *progress.Service) *Service {
	return &Service{Store: store, Invoices: inv, Progress: prog}
}

// CreateBuy starts an Order and attaches an LN invoice.
func (s *Service) CreateBuy(ctx context.Context, playerID, sku string, now time.Time) (persist.Order, error) {
	item, ok := catalog.Lookup(sku)
	if !ok || item.Sats <= 0 {
		return persist.Order{}, fmt.Errorf("unknown shop sku")
	}
	if item.Kind == catalog.KindPremium {
		se, err := s.Store.CurrentSeason(now)
		if err != nil {
			return persist.Order{}, err
		}
		prog, err := s.Store.GetOrCreateProgress(playerID, se.ID)
		if err != nil {
			return persist.Order{}, err
		}
		if prog.PremiumUnlocked {
			return persist.Order{}, persist.ErrAlreadyOwns
		}
	}
	id, err := newOrderID()
	if err != nil {
		return persist.Order{}, err
	}
	o := persist.Order{
		ID:        id,
		PlayerID:  playerID,
		SKU:       sku,
		Sats:      item.Sats,
		State:     persist.OrderCreated,
		CreatedAt: now.UTC(),
		ExpiresAt: now.UTC().Add(InvoiceTTL),
	}
	if err := s.Store.SaveOrder(o); err != nil {
		return persist.Order{}, err
	}
	inv, err := s.Invoices.CreateInbound(ctx, item.Sats, "gotype · "+item.Name, id, int(InvoiceTTL.Seconds()))
	if err != nil {
		o.State = persist.OrderFailed
		_ = s.Store.SaveOrder(o)
		return persist.Order{}, err
	}
	o.State = persist.OrderInvoiced
	o.Bolt11 = inv.PaymentRequest
	o.PaymentHash = inv.PaymentHash
	o.CheckingID = inv.CheckingID
	if err := s.Store.SaveOrder(o); err != nil {
		return persist.Order{}, err
	}
	return o, nil
}

// PollAndGrant confirms payment via poll and grants Inventory once.
func (s *Service) PollAndGrant(ctx context.Context, orderID string, now time.Time) (persist.Order, error) {
	o, err := s.Store.GetOrder(orderID)
	if err != nil {
		return persist.Order{}, err
	}
	if o.State == persist.OrderGranted {
		return o, nil
	}
	if o.State != persist.OrderInvoiced && o.State != persist.OrderPaid {
		return o, persist.ErrBadState
	}
	st, err := s.Invoices.CheckPaid(ctx, o.CheckingID)
	if err != nil {
		return o, err
	}
	if !st.Paid {
		// Late pay still ok after ExpiresAt if hash matches — keep invoiced for poll/reconcile.
		return o, nil
	}
	if o.PaymentHash != "" && st.PaymentHash != "" && st.PaymentHash != o.PaymentHash {
		return o, fmt.Errorf("payment_hash mismatch")
	}
	o.State = persist.OrderPaid
	if err := s.Store.SaveOrder(o); err != nil {
		return o, err
	}
	return s.grantLocked(o, now)
}

func (s *Service) grantLocked(o persist.Order, now time.Time) (persist.Order, error) {
	if o.State == persist.OrderGranted {
		return o, nil
	}
	item, ok := catalog.Lookup(o.SKU)
	if !ok {
		o.State = persist.OrderFailed
		_ = s.Store.SaveOrder(o)
		return o, fmt.Errorf("unknown sku on grant")
	}
	switch item.Kind {
	case catalog.KindConsumable:
		if err := s.Store.AddInventory(o.PlayerID, o.SKU, 1); err != nil {
			return o, err
		}
	case catalog.KindPremium:
		if s.Progress == nil {
			return o, fmt.Errorf("progress service required")
		}
		if err := s.Progress.UnlockPremium(o.PlayerID, now); err != nil {
			if err == persist.ErrAlreadyOwns {
				// still mark granted — payment succeeded
			} else {
				return o, err
			}
		}
	default:
		return o, fmt.Errorf("sku not grantable via shop")
	}
	o.State = persist.OrderGranted
	o.GrantedAt = now.UTC()
	if err := s.Store.SaveOrder(o); err != nil {
		return o, err
	}
	return o, nil
}

func newOrderID() (string, error) {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return fmt.Sprintf("ord_%x", b[:]), nil
}
