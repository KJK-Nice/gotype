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
