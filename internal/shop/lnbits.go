package shop

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// LNBitsConfig from environment.
type LNBitsConfig struct {
	BaseURL string // e.g. https://legend.lnbits.com
	APIKey  string // invoice/read key
}

// LNBitsFromEnv reads LNBITS_URL + LNBITS_API_KEY (invoice key).
func LNBitsFromEnv() LNBitsConfig {
	return LNBitsConfig{
		BaseURL: strings.TrimRight(strings.TrimSpace(os.Getenv("LNBITS_URL")), "/"),
		APIKey:  strings.TrimSpace(os.Getenv("LNBITS_API_KEY")),
	}
}

// Configured is true when shop LNBits credentials are set.
func (c LNBitsConfig) Configured() bool {
	return c.BaseURL != "" && c.APIKey != ""
}

// InvoiceClient creates and polls LNBits inbound invoices.
type InvoiceClient struct {
	Cfg    LNBitsConfig
	Client *http.Client
}

func NewInvoiceClient(cfg LNBitsConfig) *InvoiceClient {
	return &InvoiceClient{
		Cfg:    cfg,
		Client: &http.Client{Timeout: 15 * time.Second},
	}
}

// CreatedInvoice is the LNBits create response subset we need.
type CreatedInvoice struct {
	PaymentHash    string
	PaymentRequest string
	CheckingID     string
}

type createBody struct {
	Out        bool           `json:"out"`
	Amount     int            `json:"amount"`
	Memo       string         `json:"memo,omitempty"`
	Expiry     int            `json:"expiry,omitempty"`
	Extra      map[string]any `json:"extra,omitempty"`
	ExternalID string         `json:"external_id,omitempty"`
	Webhook    string         `json:"webhook,omitempty"`
}

// CreateInbound creates a receive invoice (out: false).
func (c *InvoiceClient) CreateInbound(ctx context.Context, sats int, memo, orderID string, expirySec int) (CreatedInvoice, error) {
	if !c.Cfg.Configured() {
		return CreatedInvoice{}, fmt.Errorf("set LNBITS_URL and LNBITS_API_KEY")
	}
	if sats <= 0 {
		return CreatedInvoice{}, fmt.Errorf("sats must be positive")
	}
	if expirySec <= 0 {
		expirySec = 15 * 60
	}
	body := createBody{
		Out:        false,
		Amount:     sats,
		Memo:       memo,
		Expiry:     expirySec,
		Extra:      map[string]any{"order_id": orderID},
		ExternalID: orderID,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return CreatedInvoice{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Cfg.BaseURL+"/api/v1/payments", bytes.NewReader(raw))
	if err != nil {
		return CreatedInvoice{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", c.Cfg.APIKey)
	res, err := c.Client.Do(req)
	if err != nil {
		return CreatedInvoice{}, err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return CreatedInvoice{}, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return CreatedInvoice{}, fmt.Errorf("lnbits create: http %s: %s", res.Status, truncate(string(data), 160))
	}
	var resp struct {
		PaymentHash    string `json:"payment_hash"`
		PaymentRequest string `json:"payment_request"`
		CheckingID     string `json:"checking_id"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return CreatedInvoice{}, err
	}
	if resp.PaymentRequest == "" || resp.CheckingID == "" {
		return CreatedInvoice{}, fmt.Errorf("lnbits create: missing invoice fields")
	}
	return CreatedInvoice{
		PaymentHash:    resp.PaymentHash,
		PaymentRequest: resp.PaymentRequest,
		CheckingID:     resp.CheckingID,
	}, nil
}

// PaymentStatus is a poll result.
type PaymentStatus struct {
	Paid        bool
	Status      string
	PaymentHash string
}

// CheckPaid polls GET /api/v1/payments/{checking_id}.
func (c *InvoiceClient) CheckPaid(ctx context.Context, checkingID string) (PaymentStatus, error) {
	if !c.Cfg.Configured() {
		return PaymentStatus{}, fmt.Errorf("set LNBITS_URL and LNBITS_API_KEY")
	}
	if checkingID == "" {
		return PaymentStatus{}, fmt.Errorf("empty checking_id")
	}
	url := c.Cfg.BaseURL + "/api/v1/payments/" + checkingID
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return PaymentStatus{}, err
	}
	req.Header.Set("X-Api-Key", c.Cfg.APIKey)
	res, err := c.Client.Do(req)
	if err != nil {
		return PaymentStatus{}, err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return PaymentStatus{}, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return PaymentStatus{}, fmt.Errorf("lnbits check: http %s: %s", res.Status, truncate(string(data), 160))
	}
	var resp struct {
		Paid        bool   `json:"paid"`
		PaymentHash string `json:"payment_hash"`
		Details     struct {
			Status      string `json:"status"`
			PaymentHash string `json:"payment_hash"`
		} `json:"details"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return PaymentStatus{}, err
	}
	paid := resp.Paid || resp.Details.Status == "success"
	hash := resp.PaymentHash
	if hash == "" {
		hash = resp.Details.PaymentHash
	}
	return PaymentStatus{Paid: paid, Status: resp.Details.Status, PaymentHash: hash}, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
