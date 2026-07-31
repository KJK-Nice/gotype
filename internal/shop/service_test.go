package shop

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/kjkusap/monkeytype-clone/internal/catalog"
	"github.com/kjkusap/monkeytype-clone/internal/persist"
	"github.com/kjkusap/monkeytype-clone/internal/player"
	"github.com/kjkusap/monkeytype-clone/internal/progress"
)

type fakeInv struct {
	paid bool
	n    int
}

func (f *fakeInv) CreateInbound(_ context.Context, sats int, memo, orderID string, expirySec int) (CreatedInvoice, error) {
	return CreatedInvoice{
		PaymentHash:    "h",
		PaymentRequest: "lnbc1fake",
		CheckingID:     "c1",
	}, nil
}

func (f *fakeInv) CheckPaid(_ context.Context, checkingID string) (PaymentStatus, error) {
	f.n++
	return PaymentStatus{Paid: f.paid, Status: "success", PaymentHash: "h"}, nil
}

func TestBuyGrantIdempotent(t *testing.T) {
	dir := t.TempDir()
	store, err := persist.Open(filepath.Join(dir, "gotype.json"))
	if err != nil {
		t.Fatal(err)
	}
	ps := player.NewService(store)
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	reg, err := ps.Register("Buyer1", "1.1.1.1", "s", now)
	if err != nil {
		t.Fatal(err)
	}
	prog := progress.NewService(store)
	inv := &fakeInv{paid: false}
	svc := NewService(store, inv, prog)

	o, err := svc.CreateBuy(context.Background(), reg.Player.ID, catalog.SKUHeart, now)
	if err != nil {
		t.Fatal(err)
	}
	if o.State != persist.OrderInvoiced || o.Bolt11 == "" {
		t.Fatalf("%+v", o)
	}
	o2, err := svc.PollAndGrant(context.Background(), o.ID, now)
	if err != nil {
		t.Fatal(err)
	}
	if o2.State != persist.OrderInvoiced {
		t.Fatalf("unpaid state=%s", o2.State)
	}
	inv.paid = true
	o3, err := svc.PollAndGrant(context.Background(), o.ID, now)
	if err != nil {
		t.Fatal(err)
	}
	if o3.State != persist.OrderGranted {
		t.Fatalf("state=%s", o3.State)
	}
	if store.InventoryQty(reg.Player.ID, catalog.SKUHeart) != 1 {
		t.Fatal("expected heart qty 1")
	}
	o4, err := svc.PollAndGrant(context.Background(), o.ID, now)
	if err != nil {
		t.Fatal(err)
	}
	if o4.State != persist.OrderGranted {
		t.Fatal("idempotent grant")
	}
	if store.InventoryQty(reg.Player.ID, catalog.SKUHeart) != 1 {
		t.Fatal("double grant")
	}
}
