package shop

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// PhoenixdConfig from environment.
type PhoenixdConfig struct {
	BaseURL    string // e.g. http://phoenixd.railway.internal:9740
	Password   string // HTTP API password (Basic auth, empty username)
	WebhookURL string // per-invoice webhookUrl (empty = omit)
}

// PhoenixdFromEnv reads PHOENIXD_URL + PHOENIXD_PASSWORD (or PHOENIXD_API_PASSWORD)
// and optional GOTYPE_WEBHOOK_URL for payment_received callbacks.
func PhoenixdFromEnv() PhoenixdConfig {
	pass := strings.TrimSpace(os.Getenv("PHOENIXD_PASSWORD"))
	if pass == "" {
		pass = strings.TrimSpace(os.Getenv("PHOENIXD_API_PASSWORD"))
	}
	return PhoenixdConfig{
		BaseURL:    strings.TrimRight(strings.TrimSpace(os.Getenv("PHOENIXD_URL")), "/"),
		Password:   pass,
		WebhookURL: strings.TrimSpace(os.Getenv("GOTYPE_WEBHOOK_URL")),
	}
}

// Configured is true when Phoenixd credentials are set.
func (c PhoenixdConfig) Configured() bool {
	return c.BaseURL != "" && c.Password != ""
}

// PhoenixdClient creates and polls Phoenixd inbound invoices.
type PhoenixdClient struct {
	Cfg    PhoenixdConfig
	Client *http.Client
}

func NewPhoenixdClient(cfg PhoenixdConfig) *PhoenixdClient {
	return &PhoenixdClient{
		Cfg:    cfg,
		Client: &http.Client{Timeout: 15 * time.Second},
	}
}

// CreateInbound POSTs /createinvoice (form-urlencoded, Basic auth).
// externalID is stored as Phoenixd externalId (Order id or Tip id).
// When Cfg.WebhookURL is set, it is passed as webhookUrl for payment_received push.
func (c *PhoenixdClient) CreateInbound(ctx context.Context, sats int, memo, externalID string, expirySec int) (CreatedInvoice, error) {
	if !c.Cfg.Configured() {
		return CreatedInvoice{}, fmt.Errorf("set PHOENIXD_URL and PHOENIXD_PASSWORD")
	}
	if sats <= 0 {
		return CreatedInvoice{}, fmt.Errorf("sats must be positive")
	}
	if expirySec <= 0 {
		expirySec = 15 * 60
	}
	form := url.Values{}
	form.Set("amountSat", fmt.Sprintf("%d", sats))
	form.Set("description", memo)
	form.Set("expirySeconds", fmt.Sprintf("%d", expirySec))
	if externalID != "" {
		form.Set("externalId", externalID)
	}
	if c.Cfg.WebhookURL != "" {
		form.Set("webhookUrl", c.Cfg.WebhookURL)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Cfg.BaseURL+"/createinvoice", strings.NewReader(form.Encode()))
	if err != nil {
		return CreatedInvoice{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth("", c.Cfg.Password)
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
		return CreatedInvoice{}, fmt.Errorf("phoenixd create: http %s: %s", res.Status, truncate(string(data), 160))
	}
	var resp struct {
		PaymentHash string `json:"paymentHash"`
		Serialized  string `json:"serialized"`
		AmountSat   int    `json:"amountSat"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return CreatedInvoice{}, err
	}
	if resp.Serialized == "" || resp.PaymentHash == "" {
		return CreatedInvoice{}, fmt.Errorf("phoenixd create: missing invoice fields")
	}
	return CreatedInvoice{
		PaymentHash:    resp.PaymentHash,
		PaymentRequest: resp.Serialized,
		CheckingID:     resp.PaymentHash,
	}, nil
}

// CheckPaid GETs /payments/incoming/{paymentHash}. 404 => unpaid.
func (c *PhoenixdClient) CheckPaid(ctx context.Context, checkingID string) (PaymentStatus, error) {
	if !c.Cfg.Configured() {
		return PaymentStatus{}, fmt.Errorf("set PHOENIXD_URL and PHOENIXD_PASSWORD")
	}
	if checkingID == "" {
		return PaymentStatus{}, fmt.Errorf("empty payment hash")
	}
	url := c.Cfg.BaseURL + "/payments/incoming/" + checkingID
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return PaymentStatus{}, err
	}
	req.SetBasicAuth("", c.Cfg.Password)
	res, err := c.Client.Do(req)
	if err != nil {
		return PaymentStatus{}, err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return PaymentStatus{}, err
	}
	if res.StatusCode == http.StatusNotFound {
		return PaymentStatus{Paid: false, Status: "pending", PaymentHash: checkingID}, nil
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return PaymentStatus{}, fmt.Errorf("phoenixd check: http %s: %s", res.Status, truncate(string(data), 160))
	}
	var resp struct {
		PaymentHash string `json:"paymentHash"`
		IsPaid      bool   `json:"isPaid"`
		ReceivedSat int    `json:"receivedSat"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return PaymentStatus{}, err
	}
	hash := resp.PaymentHash
	if hash == "" {
		hash = checkingID
	}
	status := "pending"
	if resp.IsPaid {
		status = "success"
	}
	return PaymentStatus{Paid: resp.IsPaid, Status: status, PaymentHash: hash}, nil
}
