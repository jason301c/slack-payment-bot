package main

import (
	"fmt"
	"log"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/paymentlink"
	"github.com/stripe/stripe-go/v82/price"
	"github.com/stripe/stripe-go/v82/product"
)

// StripeGenerator implements PaymentLinkGenerator for Stripe
type StripeGenerator struct {
	apiKey string
}

// NewStripeGenerator creates a new Stripe payment link generator
func NewStripeGenerator(apiKey string) PaymentLinkGenerator {
	return &StripeGenerator{
		apiKey: apiKey,
	}
}

// GenerateLink creates a Stripe payment link (one-time or recurring)
func (s *StripeGenerator) GenerateLink(data *PaymentLinkData) (string, error) {
	stripe.Key = s.apiKey

	// Create a product
	productParams := &stripe.ProductParams{
		Name:        stripe.String(data.ServiceName),
		Description: stripe.String(data.ReferenceNumber),
	}
	product, err := product.New(productParams)
	if err != nil {
		log.Printf("Stripe product error: %v", err)
		return "", fmt.Errorf("failed to create Stripe product: %w", err)
	}

	// Create a price (recurring or one-time)
	priceParams := s.buildPriceParams(data, product.ID)
	price, err := price.New(priceParams)
	if err != nil {
		log.Printf("Stripe price error: %v", err)
		return "", fmt.Errorf("failed to create Stripe price: %w", err)
	}

	// Create a payment link
	linkParams := s.buildPaymentLinkParams(data, price.ID)
	link, err := paymentlink.New(linkParams)
	if err != nil {
		log.Printf("Stripe payment link error: %v", err)
		return "", fmt.Errorf("failed to create Stripe payment link: %w", err)
	}

	log.Printf("Successfully created Stripe payment link: %s", link.URL)
	return link.URL, nil
}

// buildPriceParams constructs Stripe price parameters based on payment data
func (s *StripeGenerator) buildPriceParams(data *PaymentLinkData, productID string) *stripe.PriceParams {
	priceParams := &stripe.PriceParams{
		Currency:   stripe.String("usd"),
		UnitAmount: stripe.Int64(int64(data.Amount * 100)), // Convert to cents
		Product:    stripe.String(productID),
	}

	// Add recurring parameters for subscriptions
	if data.IsSubscription {
		interval := data.Interval
		if interval == "" {
			interval = "month" // Default to monthly
		}
		intervalCount := data.IntervalCount
		if intervalCount <= 0 {
			intervalCount = 1 // Default to every interval
		}

		priceParams.Recurring = &stripe.PriceRecurringParams{
			Interval:      stripe.String(interval),
			IntervalCount: stripe.Int64(intervalCount),
		}
	}

	return priceParams
}

// buildPaymentLinkParams constructs Stripe payment link parameters
func (s *StripeGenerator) buildPaymentLinkParams(data *PaymentLinkData, priceID string) *stripe.PaymentLinkParams {
	params := &stripe.PaymentLinkParams{
		LineItems: []*stripe.PaymentLinkLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(1),
			},
		},
	}

	// For one-time payments, save card for future use
	if !data.IsSubscription {
		params.PaymentIntentData = &stripe.PaymentLinkPaymentIntentDataParams{
			SetupFutureUsage: stripe.String("off_session"),
		}
	}

	return params
}
