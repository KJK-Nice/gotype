package ln

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

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

	t.Setenv("TIP_LIGHTNING_ADDRESS", "")
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
	t.Setenv("TIP_LIGHTNING_ADDRESS", "")
	t.Setenv("TIP_LNURL", "")
	if Configured() {
		t.Fatal("expected false")
	}
	t.Setenv("TIP_LIGHTNING_ADDRESS", "a@b.com")
	if !Configured() {
		t.Fatal("expected true")
	}
}
