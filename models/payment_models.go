package models

// PaymentLinkData represents the data needed to create a payment link
type PaymentLinkData struct {
	Amount          float64 `json:"amount"`
	ServiceName     string  `json:"service_name"`
	ReferenceNumber string  `json:"reference_number"`
	IsSubscription  bool    `json:"is_subscription"`
	Interval        string  `json:"interval"`       // e.g. "month", "week", "year"
	IntervalCount   int64   `json:"interval_count"` // e.g. 1 for every month, 3 for every 3 months
}

// PaymentProvider represents the payment service provider
type PaymentProvider string

const (
	ProviderStripe    PaymentProvider = "stripe"
	ProviderAirwallex PaymentProvider = "airwallex"
)
