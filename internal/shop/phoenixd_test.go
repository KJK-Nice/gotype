package shop

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPhoenixdCreateAndPoll(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/createinvoice", func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "" || pass != "secret" {
			http.Error(w, "auth", 401)
			return
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.Form.Get("amountSat") != "21" {
			t.Errorf("amountSat=%q", r.Form.Get("amountSat"))
		}
		if r.Form.Get("externalId") != "ord_1" {
			t.Errorf("externalId=%q", r.Form.Get("externalId"))
		}
		if r.Form.Get("webhookUrl") != "https://example.com/ln/webhook" {
			t.Errorf("webhookUrl=%q", r.Form.Get("webhookUrl"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"amountSat":   21,
			"paymentHash": "hash1",
			"serialized":  "lnbc21n1test",
		})
	})
	mux.HandleFunc("/payments/incoming/hash1", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"paymentHash": "hash1",
			"isPaid":      true,
			"receivedSat": 21,
		})
	})
	mux.HandleFunc("/payments/incoming/pending", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := NewPhoenixdClient(PhoenixdConfig{
		BaseURL:    srv.URL,
		Password:   "secret",
		WebhookURL: "https://example.com/ln/webhook",
	})
	inv, err := c.CreateInbound(context.Background(), 21, "test", "ord_1", 900)
	if err != nil {
		t.Fatal(err)
	}
	if inv.PaymentRequest != "lnbc21n1test" || inv.CheckingID != "hash1" {
		t.Fatalf("%+v", inv)
	}
	st, err := c.CheckPaid(context.Background(), "hash1")
	if err != nil || !st.Paid {
		t.Fatalf("paid=%v err=%v", st.Paid, err)
	}
	st, err = c.CheckPaid(context.Background(), "pending")
	if err != nil || st.Paid {
		t.Fatalf("pending want unpaid, got paid=%v err=%v", st.Paid, err)
	}
}
