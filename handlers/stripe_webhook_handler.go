package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/subscription"
	"github.com/stripe/stripe-go/v82/webhook"
)

// StripeWebhookHandler handles Stripe webhook events
type StripeWebhookHandler struct {
	endpointSecret string
	stripeAPIKey   string
}

// NewStripeWebhookHandler creates a new Stripe webhook handler
func NewStripeWebhookHandler(endpointSecret, stripeAPIKey string) *StripeWebhookHandler {
	return &StripeWebhookHandler{
		endpointSecret: endpointSecret,
		stripeAPIKey:   stripeAPIKey,
	}
}

// HandleWebhook processes incoming Stripe webhook events
func (h *StripeWebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	const MaxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading webhook payload: %v", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	// Verify webhook signature
	event, err := webhook.ConstructEvent(payload, r.Header.Get("Stripe-Signature"), h.endpointSecret)
	if err != nil {
		log.Printf("Error verifying webhook signature: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Handle the event
	switch event.Type {
	case "checkout.session.completed":
		h.handleCheckoutSessionCompleted(event)
	case "customer.subscription.created":
		h.handleSubscriptionCreated(event)
	default:
		log.Printf("Unhandled event type: %s", event.Type)
	}

	w.WriteHeader(http.StatusOK)
}

// handleCheckoutSessionCompleted processes successful checkout sessions
func (h *StripeWebhookHandler) handleCheckoutSessionCompleted(event stripe.Event) {
	var session stripe.CheckoutSession
	err := json.Unmarshal(event.Data.Raw, &session)
	if err != nil {
		log.Printf("Error parsing checkout session: %v", err)
		return
	}

	log.Printf("Checkout session completed: %s", session.ID)

	// If this was a subscription checkout, the subscription will be created separately
	// and handled in handleSubscriptionCreated
}

// handleSubscriptionCreated processes new subscription events and schedules cancellation if needed
func (h *StripeWebhookHandler) handleSubscriptionCreated(event stripe.Event) {
	var sub stripe.Subscription
	err := json.Unmarshal(event.Data.Raw, &sub)
	if err != nil {
		log.Printf("[Webhook] Error parsing subscription: %v", err)
		return
	}

	log.Printf("[Webhook] Subscription created: %s (Customer: %s, Status: %s)", sub.ID, sub.Customer.ID, sub.Status)
	log.Printf("[Webhook] Subscription metadata: %+v", sub.Metadata)

	// Check if this subscription has cycle limits in metadata
	if endCyclesStr, exists := sub.Metadata["end_date_cycles"]; exists {
		log.Printf("[Webhook] Found EndDateCycles in subscription %s metadata", sub.ID)
		
		endTimestampStr, timestampExists := sub.Metadata["end_timestamp"]
		if !timestampExists {
			log.Printf("[Webhook] ERROR: Subscription %s has end_date_cycles but no end_timestamp", sub.ID)
			return
		}

		interval := sub.Metadata["interval"]
		intervalCount := sub.Metadata["interval_count"]
		serviceName := sub.Metadata["service_name"]
		
		log.Printf("[Webhook] Subscription details - Service: %s, Interval: %s, Count: %s", serviceName, interval, intervalCount)

		endCycles, err := strconv.ParseInt(endCyclesStr, 10, 64)
		if err != nil {
			log.Printf("[Webhook] ERROR: Error parsing end_date_cycles for subscription %s: %v", sub.ID, err)
			return
		}

		endTimestamp, err := strconv.ParseInt(endTimestampStr, 10, 64)
		if err != nil {
			log.Printf("[Webhook] ERROR: Error parsing end_timestamp for subscription %s: %v", sub.ID, err)
			return
		}

		endTime := time.Unix(endTimestamp, 0)
		log.Printf("[Webhook] Scheduling subscription %s to cancel after %d cycles", sub.ID, endCycles)
		log.Printf("[Webhook] Cancellation scheduled for: %s (timestamp: %d)", endTime.Format("2006-01-02 15:04:05 UTC"), endTimestamp)

		// Schedule the subscription to cancel at the calculated end time
		err = h.scheduleSubscriptionCancellation(sub.ID, endTimestamp)
		if err != nil {
			log.Printf("[Webhook] ERROR: Failed to schedule cancellation for subscription %s: %v", sub.ID, err)
			return
		}

		log.Printf("[Webhook] ✅ Successfully scheduled cancellation for subscription %s", sub.ID)
	} else {
		log.Printf("[Webhook] Subscription %s has no EndDateCycles - will run indefinitely", sub.ID)
	}
}

// scheduleSubscriptionCancellation sets a subscription to cancel at a specific timestamp
func (h *StripeWebhookHandler) scheduleSubscriptionCancellation(subscriptionID string, cancelAtTimestamp int64) error {
	log.Printf("[Webhook] Setting Stripe API key and preparing cancellation params for subscription %s", subscriptionID)
	stripe.Key = h.stripeAPIKey

	params := &stripe.SubscriptionParams{
		CancelAt:          stripe.Int64(cancelAtTimestamp),
		CancelAtPeriodEnd: stripe.Bool(true),
	}

	log.Printf("[Webhook] Calling Stripe API to update subscription %s with cancellation params", subscriptionID)
	updatedSub, err := subscription.Update(subscriptionID, params)
	if err != nil {
		log.Printf("[Webhook] ERROR: Stripe API call failed for subscription %s: %v", subscriptionID, err)
		return fmt.Errorf("failed to schedule subscription cancellation: %w", err)
	}

	cancelTime := time.Unix(cancelAtTimestamp, 0)
	log.Printf("[Webhook] ✅ Stripe API call successful - subscription %s will cancel at %s", 
		subscriptionID, cancelTime.Format("2006-01-02 15:04:05 UTC"))
	log.Printf("[Webhook] Updated subscription status: %s, cancel_at_period_end: %t", 
		updatedSub.Status, updatedSub.CancelAtPeriodEnd)

	return nil
}
