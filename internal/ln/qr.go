package ln

import (
	"bytes"
	"strings"

	"github.com/mdp/qrterminal/v3"
)

// QRString renders a compact half-block QR for terminal display.
func QRString(payload string) string {
	var buf bytes.Buffer
	qrterminal.GenerateHalfBlock(payload, qrterminal.L, &buf)
	return strings.TrimRight(buf.String(), "\n")
}

// ShortBolt11 truncates an invoice for on-screen display.
func ShortBolt11(pr string) string {
	pr = strings.TrimSpace(pr)
	if len(pr) <= 28 {
		return pr
	}
	return pr[:14] + "…" + pr[len(pr)-10:]
}
