package shop

// CreatedInvoice is returned by any Invoicer after create.
type CreatedInvoice struct {
	PaymentHash    string
	PaymentRequest string
	CheckingID     string // poll key; for Phoenixd equals payment hash
}

// PaymentStatus is a poll result.
type PaymentStatus struct {
	Paid        bool
	Status      string
	PaymentHash string
}
