package shop

// NewInvoicerFromEnv picks Phoenixd (preferred) or LNBits.
func NewInvoicerFromEnv() Invoicer {
	if cfg := PhoenixdFromEnv(); cfg.Configured() {
		return NewPhoenixdClient(cfg)
	}
	return NewInvoiceClient(LNBitsFromEnv())
}

// ShopConfigured is true when any shop invoice backend is set.
func ShopConfigured() bool {
	return PhoenixdFromEnv().Configured() || LNBitsFromEnv().Configured()
}
