package shop

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLNBitsCreateAndPoll(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/payments", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != "inkey" {
			http.Error(w, "auth", 401)
			return
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["out"] != false {
			t.Errorf("out = %v, want false", body["out"])
		}
		w.WriteHeader(201)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"payment_hash":    "hash1",
			"payment_request": "lnbc21n1test",
			"checking_id":     "chk1",
		})
	})
	mux.HandleFunc("/api/v1/payments/chk1", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"paid":         true,
			"payment_hash": "hash1",
			"details":      map[string]string{"status": "success", "payment_hash": "hash1"},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := NewInvoiceClient(LNBitsConfig{BaseURL: srv.URL, APIKey: "inkey"})
	inv, err := c.CreateInbound(context.Background(), 21, "test", "ord_1", 900)
	if err != nil {
		t.Fatal(err)
	}
	if inv.PaymentRequest != "lnbc21n1test" || inv.CheckingID != "chk1" {
		t.Fatalf("%+v", inv)
	}
	st, err := c.CheckPaid(context.Background(), "chk1")
	if err != nil || !st.Paid {
		t.Fatalf("paid=%v err=%v", st.Paid, err)
	}
}
