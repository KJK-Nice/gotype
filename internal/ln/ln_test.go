package ln

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/kjkusap/monkeytype-clone/internal/persist"
)

func clearTipEnv(t *testing.T) {
	t.Helper()
	t.Setenv("PHOENIXD_URL", "")
	t.Setenv("PHOENIXD_PASSWORD", "")
	t.Setenv("PHOENIXD_API_PASSWORD", "")
	t.Setenv("GOTYPE_WEBHOOK_URL", "")
	t.Setenv("TIP_LIGHTNING_ADDRESS", "")
	t.Setenv("TIP_LNURL", "")
}

func TestCreateInvoicePhoenixdTracked(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/createinvoice", func(w http.ResponseWriter, r *http.Request) {
		_, pass, ok := r.BasicAuth()
		if !ok || pass != "secret" {
			http.Error(w, "auth", 401)
			return
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.Form.Get("amountSat") != "21" {
			t.Errorf("amountSat=%q", r.Form.Get("amountSat"))
		}
		if r.Form.Get("description") != "gotype tip" {
			t.Errorf("description=%q", r.Form.Get("description"))
		}
		if !stringsHasPrefix(r.Form.Get("externalId"), "tip_") {
			t.Errorf("externalId=%q", r.Form.Get("externalId"))
		}
		if r.Form.Get("webhookUrl") != "https://gotype.fun/ln/webhook" {
			t.Errorf("webhookUrl=%q", r.Form.Get("webhookUrl"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"amountSat":   21,
			"paymentHash": "hash1",
			"serialized":  "lnbc21n1phoenixtip",
		})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	store, err := persist.Open(filepath.Join(dir, "data.json"))
	if err != nil {
		t.Fatal(err)
	}

	clearTipEnv(t)
	t.Setenv("PHOENIXD_URL", srv.URL)
	t.Setenv("PHOENIXD_PASSWORD", "secret")
	t.Setenv("GOTYPE_WEBHOOK_URL", "https://gotype.fun/ln/webhook")
	t.Setenv("TIP_LIGHTNING_ADDRESS", "ignored@walletofsatoshi.com")

	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	inv, err := CreateInvoice(context.Background(), store, 21, "gotype tip", now)
	if err != nil {
		t.Fatal(err)
	}
	if inv.Sats != 21 || inv.Bolt11 != "lnbc21n1phoenixtip" || !inv.Tracked || inv.ID == "" {
		t.Fatalf("inv=%+v", inv)
	}
	tip, err := store.GetTip(inv.ID)
	if err != nil || tip.State != persist.TipPending || tip.PaymentHash != "hash1" {
		t.Fatalf("tip=%+v err=%v", tip, err)
	}
	if Destination() != "phoenixd" {
		t.Fatalf("destination=%q", Destination())
	}
}

func stringsHasPrefix(s, p string) bool {
	return len(s) >= len(p) && s[:len(p)] == p
}

func TestPollTipMarksPaid(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/payments/incoming/hash1", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"paymentHash": "hash1",
			"isPaid":      true,
			"receivedSat": 21,
		})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	store, err := persist.Open(filepath.Join(dir, "data.json"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	tip := persist.TipIntent{
		ID: "tip_1", Sats: 21, State: persist.TipPending,
		PaymentHash: "hash1", CheckingID: "hash1", CreatedAt: now,
	}
	if err := store.SaveTip(tip); err != nil {
		t.Fatal(err)
	}

	clearTipEnv(t)
	t.Setenv("PHOENIXD_URL", srv.URL)
	t.Setenv("PHOENIXD_PASSWORD", "secret")

	got, err := PollTip(context.Background(), store, tip.ID, now)
	if err != nil || got.State != persist.TipPaid {
		t.Fatalf("got=%+v err=%v", got, err)
	}
	got2, err := PollTip(context.Background(), store, tip.ID, now)
	if err != nil || got2.State != persist.TipPaid {
		t.Fatal("idempotent")
	}
}

func TestCreateInvoiceLNURLPay(t *testing.T) {
	var sawAmount string
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/lnurlp/tipjar", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"tag":             "payRequest",
			"callback":        "http://" + r.Host + "/cb",
			"minSendable":     1000,
			"maxSendable":     5_000_000,
			"commentAllowed":  40,
			"metadata":        "[[\"text/plain\",\"mtype tips\"]]",
		})
	})
	mux.HandleFunc("/cb", func(w http.ResponseWriter, r *http.Request) {
		sawAmount = r.URL.Query().Get("amount")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"pr": "lnbc21u1testinvoicepayload",
		})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	clearTipEnv(t)
	t.Setenv("TIP_LNURL", srv.URL+"/.well-known/lnurlp/tipjar")

	inv, err := CreateInvoice(context.Background(), nil, 21, "mtype tip", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if inv.Sats != 21 {
		t.Fatalf("sats=%d", inv.Sats)
	}
	if inv.Bolt11 == "" || inv.Tracked {
		t.Fatalf("inv=%+v", inv)
	}
	if sawAmount != "21000" {
		t.Fatalf("amount msats=%q want 21000", sawAmount)
	}
}

func TestResolveLightningAddress(t *testing.T) {
	u, err := resolvePayURL("alice@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if u != "https://example.com/.well-known/lnurlp/alice" {
		t.Fatalf("got %s", u)
	}
}

func TestConfigured(t *testing.T) {
	clearTipEnv(t)
	if Configured() {
		t.Fatal("expected false")
	}
	t.Setenv("PHOENIXD_URL", "http://127.0.0.1:9740")
	t.Setenv("PHOENIXD_PASSWORD", "secret")
	if !Configured() {
		t.Fatal("expected true with phoenixd")
	}
	clearTipEnv(t)
	t.Setenv("TIP_LIGHTNING_ADDRESS", "a@b.com")
	if !Configured() {
		t.Fatal("expected true with lightning address")
	}
}
