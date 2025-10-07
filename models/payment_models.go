package models

// PaymentLinkData represents the data needed to create a payment link
type PaymentLinkData struct {
	Amount            float64 `json:"amount"`
	ServiceName       string  `json:"service_name"`
	ReferenceNumber   string  `json:"reference_number"`
	IsSubscription    bool    `json:"is_subscription"`
	Interval          string  `json:"interval"`           // e.g. "month", "week", "year"
	IntervalCount     int64   `json:"interval_count"`     // e.g. 1 for every month, 3 for every 3 months
	EndDateCycles     int64   `json:"end_date_cycles"`    // number of cycles before subscription ends (optional)
	InternalReference string  `json:"internal_reference"` // Airwallex internal reference (optional)
}

// PaymentProvider represents the payment service provider
type PaymentProvider string

const (
	ProviderStripe    PaymentProvider = "stripe"
	ProviderAirwallex PaymentProvider = "airwallex"
)

// InvoiceData represents the data needed to create an invoice
type InvoiceData struct {
	InvoiceNumber    string            `json:"invoice_number"`
	ClientName       string            `json:"client_name"`
	ClientAddress    string            `json:"client_address"`
	ClientEmail      string            `json:"client_email"`
	DateDue          string            `json:"date_due"`
	Currency         string            `json:"currency"` // e.g., "USD", "EUR", "HKD"
	LineItems        []InvoiceLineItem `json:"line_items"`
}

// InvoiceLineItem represents a line item in an invoice
type InvoiceLineItem struct {
	ServiceDescription string  `json:"service_description"`
	UnitPrice         float64 `json:"unit_price"`
	Quantity          int     `json:"quantity"`
}
