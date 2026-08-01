package ln

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func clearTipEnv(t *testing.T) {
	t.Helper()
	t.Setenv("PHOENIXD_URL", "")
	t.Setenv("PHOENIXD_PASSWORD", "")
	t.Setenv("PHOENIXD_API_PASSWORD", "")
	t.Setenv("TIP_LIGHTNING_ADDRESS", "")
	t.Setenv("TIP_LNURL", "")
}

func TestCreateInvoicePhoenixd(t *testing.T) {
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
		_ = json.NewEncoder(w).Encode(map[string]any{
			"amountSat":   21,
			"paymentHash": "hash1",
			"serialized":  "lnbc21n1phoenixtip",
		})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	clearTipEnv(t)
	t.Setenv("PHOENIXD_URL", srv.URL)
	t.Setenv("PHOENIXD_PASSWORD", "secret")
	t.Setenv("TIP_LIGHTNING_ADDRESS", "ignored@walletofsatoshi.com")

	inv, err := CreateInvoice(context.Background(), 21, "gotype tip")
	if err != nil {
		t.Fatal(err)
	}
	if inv.Sats != 21 || inv.Bolt11 != "lnbc21n1phoenixtip" {
		t.Fatalf("inv=%+v", inv)
	}
	if Destination() != "phoenixd" {
		t.Fatalf("destination=%q", Destination())
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
			"pr": "lnbc21u1ptestestinvoicepayload",
		})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	clearTipEnv(t)
	t.Setenv("TIP_LNURL", srv.URL+"/.well-known/lnurlp/tipjar")

	inv, err := CreateInvoice(context.Background(), 21, "mtype tip")
	if err != nil {
		t.Fatal(err)
	}
	if inv.Sats != 21 {
		t.Fatalf("sats=%d", inv.Sats)
	}
	if inv.Bolt11 == "" {
		t.Fatal("empty bolt11")
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
