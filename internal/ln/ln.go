package ln

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

	"github.com/kjkusap/monkeytype-clone/internal/shop"
)

// DefaultAmounts are tip choices in satoshis.
var DefaultAmounts = []int{21, 100, 500, 2100}

// Configured is true when Phoenixd or a tip destination is set.
func Configured() bool {
	return shop.PhoenixdFromEnv().Configured() || tipDestination() != ""
}

// Destination returns the active tip backend label or LNURL endpoint.
func Destination() string {
	if shop.PhoenixdFromEnv().Configured() {
		return "phoenixd"
	}
	return tipDestination()
}

func tipDestination() string {
	if a := strings.TrimSpace(os.Getenv("TIP_LIGHTNING_ADDRESS")); a != "" {
		return a
	}
	return strings.TrimSpace(os.Getenv("TIP_LNURL"))
}

// Invoice is a payable bolt11 tip.
type Invoice struct {
	Bolt11 string
	Sats   int
}

// CreateInvoice returns a bolt11 tip invoice via Phoenixd (preferred) or LNURL-pay.
func CreateInvoice(ctx context.Context, sats int, comment string) (Invoice, error) {
	if sats <= 0 {
		return Invoice{}, fmt.Errorf("sats must be positive")
	}
	if cfg := shop.PhoenixdFromEnv(); cfg.Configured() {
		created, err := shop.NewPhoenixdClient(cfg).CreateInbound(ctx, sats, comment, "", 15*60)
		if err != nil {
			return Invoice{}, err
		}
		return Invoice{Bolt11: created.PaymentRequest, Sats: sats}, nil
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

func resolvePayURL(dest string) (string, error) {
	dest = strings.TrimSpace(dest)
	if strings.Contains(dest, "@") && !strings.Contains(dest, "://") {
		parts := strings.Split(dest, "@")
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return "", fmt.Errorf("bad lightning address %q", dest)
		}
		user := url.PathEscape(parts[0])
		host := parts[1]
		return "https://" + host + "/.well-known/lnurlp/" + user, nil
	}
	if strings.HasPrefix(dest, "http://") || strings.HasPrefix(dest, "https://") {
		return dest, nil
	}
	return "", fmt.Errorf("unsupported tip destination (use user@host or https LNURL)")
}

type payParams struct {
	Callback        string `json:"callback"`
	MinSendable     int64  `json:"minSendable"`
	MaxSendable     int64  `json:"maxSendable"`
	CommentAllowed  int    `json:"commentAllowed"`
	Tag             string `json:"tag"`
	Status          string `json:"status"`
	Reason          string `json:"reason"`
}

func fetchPayParams(ctx context.Context, payURL string) (payParams, error) {
	var p payParams
	data, err := getJSON(ctx, payURL)
	if err != nil {
		return p, err
	}
	if err := json.Unmarshal(data, &p); err != nil {
		return p, err
	}
	if p.Status == "ERROR" {
		return p, fmt.Errorf("lnurl error: %s", p.Reason)
	}
	if p.Callback == "" {
		return p, fmt.Errorf("lnurl missing callback")
	}
	return p, nil
}

func fetchInvoice(ctx context.Context, callback string, msat int64, comment string, commentAllowed int) (string, error) {
	u, err := url.Parse(callback)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("amount", fmt.Sprintf("%d", msat))
	if comment != "" && commentAllowed > 0 {
		if len(comment) > commentAllowed {
			comment = comment[:commentAllowed]
		}
		q.Set("comment", comment)
	}
	u.RawQuery = q.Encode()

	data, err := getJSON(ctx, u.String())
	if err != nil {
		return "", err
	}
	var resp struct {
		PR     string `json:"pr"`
		Status string `json:"status"`
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", err
	}
	if resp.Status == "ERROR" {
		return "", fmt.Errorf("lnurl error: %s", resp.Reason)
	}
	if resp.PR == "" {
		return "", fmt.Errorf("empty invoice")
	}
	return resp.PR, nil
}

func getJSON(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("http %s: %s", res.Status, truncate(string(data), 120))
	}
	return data, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
