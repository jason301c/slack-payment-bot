package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/paymentlink"
	"github.com/stripe/stripe-go/v82/price"
	"github.com/stripe/stripe-go/v82/product"
)

var (
	stripeApiKey      string
	airwallexClientId string
	airwallexApiKey   string
	airwallexBaseUrl  string
)

func init() {
	stripeApiKey = os.Getenv("STRIPE_API_KEY")
	airwallexClientId = os.Getenv("AIRWALLEX_CLIENT_ID")
	airwallexApiKey = os.Getenv("AIRWALLEX_API_KEY")
	airwallexBaseUrl = os.Getenv("AIRWALLEX_BASE_URL")
	if stripeApiKey == "" {
		log.Fatal("STRIPE_API_KEY environment variable not set.")
	}
	if airwallexClientId == "" {
		log.Fatal("AIRWALLEX_CLIENT_ID environment variable not set.")
	}
	if airwallexApiKey == "" {
		log.Fatal("AIRWALLEX_API_KEY environment variable not set.")
	}
	if airwallexBaseUrl == "" {
		airwallexBaseUrl = "https://api-demo.airwallex.com" // Default to demo
	}
}

// GenerateAirwallexLink creates a real Airwallex payment link using their API.
func GenerateAirwallexLink(data *PaymentLinkData) string {
	token, err := getAirwallexToken()
	if err != nil {
		log.Printf("Airwallex auth error: %v", err)
		return "[Airwallex auth error]"
	}
	link, err := createAirwallexPaymentLink(token, data)
	if err != nil {
		log.Printf("Airwallex link error: %v", err)
		return "[Airwallex link error]"
	}
	return link
}

// getAirwallexToken authenticates and returns a bearer token.
func getAirwallexToken() (string, error) {
	form := url.Values{}
	form.Set("client_id", airwallexClientId)
	form.Set("api_key", airwallexApiKey)
	resp, err := http.PostForm(airwallexBaseUrl+"/api/v1/authentication/login", form)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var result struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.Token, nil
}

// createAirwallexPaymentLink creates a payment link via Airwallex API.
func createAirwallexPaymentLink(token string, data *PaymentLinkData) (string, error) {
	body := map[string]interface{}{
		"amount":            data.Amount,
		"currency":          "USD",
		"merchant_order_id": data.ReferenceNumber,
		"request_id":        fmt.Sprintf("slackbot-%d", time.Now().UnixNano()),
		"description":       data.ServiceName,
	}
	b, _ := json.Marshal(body)
	req, err := http.NewRequest("POST", airwallexBaseUrl+"/api/v1/pa/payment_links/create", bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var result struct {
		PaymentLinkUrl string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.PaymentLinkUrl, nil
}

// GenerateStripeLink creates a real Stripe payment link using the Stripe Go SDK.
func GenerateStripeLink(data *PaymentLinkData) string {
	stripe.Key = stripeApiKey
	ctx := context.Background()
	productParams := &stripe.ProductParams{
		Name: stripe.String(data.ServiceName),
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
		Metadata: map[string]string{
			"reference": data.ReferenceNumber,
		},
	}
	link, err := paymentlink.New(params)
	if err != nil {
		log.Printf("Stripe link error: %v", err)
		return "[Stripe link error]"
	}
	return link.URL
}
