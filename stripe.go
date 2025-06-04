package main

import (
	"log"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/paymentlink"
	"github.com/stripe/stripe-go/v82/price"
	"github.com/stripe/stripe-go/v82/product"
)

// GenerateStripeLink creates a real Stripe payment link using the Stripe Go SDK.
func GenerateStripeLink(data *PaymentLinkData) string {
	stripe.Key = stripeApiKey

	// Create a product
	productParams := &stripe.ProductParams{
		Name:        stripe.String(data.ServiceName),
		Description: stripe.String(data.ReferenceNumber),
	}
	product, err := product.New(productParams)
	if err != nil {
		log.Printf("Stripe product error: %v", err)
		return "[Stripe product error]"
	}

	// Create a price
	priceParams := &stripe.PriceParams{
		Currency:   stripe.String("usd"),
		UnitAmount: stripe.Int64(int64(data.Amount * 100)),
		Product:    stripe.String(product.ID),
	}
	price, err := price.New(priceParams)
	if err != nil {
		log.Printf("Stripe price error: %v", err)
		return "[Stripe price error]"
	}

	// Create a payment link
	params := &stripe.PaymentLinkParams{
		LineItems: []*stripe.PaymentLinkLineItemParams{
			{
				Price:    stripe.String(price.ID),
				Quantity: stripe.Int64(1),
			},
		},
		PaymentIntentData: &stripe.PaymentLinkPaymentIntentDataParams{
			SetupFutureUsage: stripe.String("off_session"), // save card for future payments
		},
	}
	link, err := paymentlink.New(params)
	if err != nil {
		log.Printf("Stripe link error: %v", err)
		return "[Stripe link error]"
	}
	return link.URL
}
