package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/paymentlink"
	"github.com/stripe/stripe-go/v82/price"
	"github.com/stripe/stripe-go/v82/product"
)

// GenerateAirwallexLink creates a real Airwallex payment link using their API.
func GenerateAirwallexLink(data *PaymentLinkData) string {
	log.Printf("[Airwallex] GenerateAirwallexLink called with: %+v", data)
	token, err := getAirwallexToken()
	if err != nil {
		log.Printf("[Airwallex] Auth error: %v", err)
		return "[Airwallex auth error]"
	}
	log.Printf("[Airwallex] Got token: %s", token)
	link, err := createAirwallexPaymentLink(token, data)
	if err != nil {
		log.Printf("[Airwallex] Link error: %v", err)
		return "[Airwallex link error]"
	}
	log.Printf("[Airwallex] Successfully created payment link: %s", link)
	return link
}

// getAirwallexToken authenticates and returns a bearer token.
func getAirwallexToken() (string, error) {
	log.Printf("[Airwallex] Authenticating with client_id=%s, api_key=%s, base_url=%s", airwallexClientId, airwallexApiKey, airwallexBaseUrl)
	url := airwallexBaseUrl + "/api/v1/authentication/login"
	body := []byte(`{}`)
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		log.Printf("[Airwallex] Error creating auth request: %v", err)
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-client-id", airwallexClientId)
	req.Header.Set("x-api-key", airwallexApiKey)
	log.Printf("[Airwallex] Sending auth request to %s with headers: x-client-id=%s, x-api-key=%s", url, airwallexClientId, airwallexApiKey)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[Airwallex] Error posting auth request: %v", err)
		return "", err
	}
	defer resp.Body.Close()
	log.Printf("[Airwallex] Auth response status: %s", resp.Status)
	respBody, _ := io.ReadAll(resp.Body)
	log.Printf("[Airwallex] Auth response body: %s", string(respBody))
	var result struct {
		Token     string `json:"token"`
		ExpiresAt string `json:"expires_at"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		log.Printf("[Airwallex] Error decoding auth response: %v", err)
		return "", err
	}
	log.Printf("[Airwallex] Received token: %s, expires_at: %s", result.Token, result.ExpiresAt)
	return result.Token, nil
}

// createAirwallexPaymentLink creates a payment link via Airwallex API.
func createAirwallexPaymentLink(token string, data *PaymentLinkData) (string, error) {
	body := map[string]interface{}{
		"amount":      data.Amount,
		"currency":    "USD",
		"title":       data.ServiceName,
		"description": data.ReferenceNumber,
		"reference":   fmt.Sprintf("slackbot-%d", time.Now().UnixNano()),
		"reusable":    false,
	}
	b, _ := json.Marshal(body)
	log.Printf("[Airwallex] Creating payment link with body: %s", string(b))
	url := airwallexBaseUrl + "/api/v1/pa/payment_links/create"
	log.Printf("[Airwallex] POST %s", url)
	req, err := http.NewRequest("POST", url, bytes.NewReader(b))
	if err != nil {
		log.Printf("[Airwallex] Error creating request: %v", err)
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[Airwallex] Error making payment link request: %v", err)
		return "", err
	}
	defer resp.Body.Close()
	log.Printf("[Airwallex] Payment link response status: %s", resp.Status)
	respBody, _ := io.ReadAll(resp.Body)
	log.Printf("[Airwallex] Payment link response body: %s", string(respBody))
	var result struct {
		PaymentLinkUrl string `json:"url"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		log.Printf("[Airwallex] Error decoding payment link response: %v", err)
		return "", err
	}
	log.Printf("[Airwallex] Received payment link: %s", result.PaymentLinkUrl)
	return result.PaymentLinkUrl, nil
}

// GenerateStripeLink creates a real Stripe payment link using the Stripe Go SDK.
func GenerateStripeLink(data *PaymentLinkData) string {
	stripe.Key = stripeApiKey
	productParams := &stripe.ProductParams{
		Name:        stripe.String(data.ServiceName),
		Description: stripe.String(data.ReferenceNumber),
	}
	product, err := product.New(productParams)
	if err != nil {
		log.Printf("Stripe product error: %v", err)
		return "[Stripe product error]"
	}
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
	params := &stripe.PaymentLinkParams{
		LineItems: []*stripe.PaymentLinkLineItemParams{
			{
				Price:    stripe.String(price.ID),
				Quantity: stripe.Int64(1),
			},
		},
		PaymentIntentData: &stripe.PaymentLinkPaymentIntentDataParams{
			SetupFutureUsage: stripe.String("off_session"),
		},
	}
	link, err := paymentlink.New(params)
	if err != nil {
		log.Printf("Stripe link error: %v", err)
		return "[Stripe link error]"
	}
	return link.URL
}
