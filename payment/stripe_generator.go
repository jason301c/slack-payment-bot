package payment

import (
	"fmt"
	"log"
	"time"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/paymentlink"
	"github.com/stripe/stripe-go/v82/price"
	"github.com/stripe/stripe-go/v82/product"

	"paymentbot/models"
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
func (s *StripeGenerator) GenerateLink(data *models.PaymentLinkData) (string, string, error) {
	stripe.Key = s.apiKey

	// Create a product
	productParams := &stripe.ProductParams{
		Name:        stripe.String(data.ServiceName),
		Description: stripe.String(data.ReferenceNumber),
	}
	product, err := product.New(productParams)
	if err != nil {
		log.Printf("Stripe product error: %v", err)
		return "", "", fmt.Errorf("failed to create Stripe product: %w", err)
	}

	// Create a price (recurring or one-time)
	priceParams := s.buildPriceParams(data, product.ID)
	price, err := price.New(priceParams)
	if err != nil {
		log.Printf("Stripe price error: %v", err)
		return "", "", fmt.Errorf("failed to create Stripe price: %w", err)
	}

	// Create a payment link
	linkParams := s.buildPaymentLinkParams(data, price.ID)
	link, err := paymentlink.New(linkParams)
	if err != nil {
		log.Printf("Stripe payment link error: %v", err)
		return "", "", fmt.Errorf("failed to create Stripe payment link: %w", err)
	}

	log.Printf("Successfully created Stripe payment link: %s (ID: %s)", link.URL, link.ID)
	return link.URL, link.ID, nil
}

// buildPriceParams constructs Stripe price parameters based on payment data
func (s *StripeGenerator) buildPriceParams(data *models.PaymentLinkData, productID string) *stripe.PriceParams {
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
func (s *StripeGenerator) buildPaymentLinkParams(data *models.PaymentLinkData, priceID string) *stripe.PaymentLinkParams {
	params := &stripe.PaymentLinkParams{
		LineItems: []*stripe.PaymentLinkLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(1),
			},
		},
	}

	// For one-time payments, enable customer creation and save card for future use
	if !data.IsSubscription {
		params.CustomerCreation = stripe.String("always")
		params.PaymentIntentData = &stripe.PaymentLinkPaymentIntentDataParams{
			SetupFutureUsage: stripe.String("off_session"),
		}
	} else {
		// For subscriptions, add metadata to track cycle limits
		log.Printf("[Stripe] Creating subscription payment link for service: %s", data.ServiceName)
		metadata := make(map[string]string)
		metadata["service_name"] = data.ServiceName
		metadata["reference_number"] = data.ReferenceNumber

		if data.EndDateCycles > 0 {
			endTimestamp := calculateEndTimestamp(data.Interval, data.IntervalCount, data.EndDateCycles)
			metadata["end_date_cycles"] = fmt.Sprintf("%d", data.EndDateCycles)
			metadata["end_timestamp"] = fmt.Sprintf("%d", endTimestamp)
			metadata["interval"] = data.Interval
			metadata["interval_count"] = fmt.Sprintf("%d", data.IntervalCount)
			
			endTime := time.Unix(endTimestamp, 0)
			log.Printf("[Stripe] Subscription will be limited to %d cycles (%s every %d %s(s))", 
				data.EndDateCycles, data.Interval, data.IntervalCount, data.Interval)
			log.Printf("[Stripe] Calculated end timestamp: %d (%s)", endTimestamp, endTime.Format("2006-01-02 15:04:05 UTC"))
			log.Printf("[Stripe] Subscription metadata: %+v", metadata)
		} else {
			log.Printf("[Stripe] Creating unlimited subscription (no EndDateCycles specified)")
		}

		params.SubscriptionData = &stripe.PaymentLinkSubscriptionDataParams{
			Metadata: metadata,
		}
	}

	return params
}

// calculateEndTimestamp calculates the Unix timestamp when subscription should end
func calculateEndTimestamp(interval string, intervalCount int64, endDateCycles int64) int64 {
	if endDateCycles <= 0 {
		return 0
	}

	now := time.Now()
	var duration time.Duration

	switch interval {
	case "day":
		duration = time.Duration(intervalCount*endDateCycles) * 24 * time.Hour
	case "week":
		duration = time.Duration(intervalCount*endDateCycles) * 7 * 24 * time.Hour
	case "month":
		// Approximate month as 30 days
		duration = time.Duration(intervalCount*endDateCycles) * 30 * 24 * time.Hour
	case "year":
		// Approximate year as 365 days
		duration = time.Duration(intervalCount*endDateCycles) * 365 * 24 * time.Hour
	default:
		// Default to month if interval is unknown
		duration = time.Duration(intervalCount*endDateCycles) * 30 * 24 * time.Hour
	}

	endTime := now.Add(duration)
	return endTime.Unix()
}
