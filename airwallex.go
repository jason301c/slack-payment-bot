package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// AirwallexGenerator implements PaymentLinkGenerator for Airwallex
type AirwallexGenerator struct {
	clientID string
	apiKey   string
	baseURL  string
	client   *http.Client
}

// NewAirwallexGenerator creates a new Airwallex payment link generator
func NewAirwallexGenerator(clientID, apiKey, baseURL string) PaymentLinkGenerator {
	return &AirwallexGenerator{
		clientID: clientID,
		apiKey:   apiKey,
		baseURL:  baseURL,
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

// GenerateLink creates an Airwallex payment link
func (a *AirwallexGenerator) GenerateLink(data *PaymentLinkData) (string, error) {
	log.Printf("[Airwallex] GenerateLink called with: %+v", data)

	// Authenticate and get token
	token, err := a.authenticate()
	if err != nil {
		log.Printf("[Airwallex] Auth error: %v", err)
		return "", fmt.Errorf("failed to authenticate with Airwallex: %w", err)
	}

	// Create payment link
	link, err := a.createPaymentLink(token, data)
	if err != nil {
		log.Printf("[Airwallex] Link creation error: %v", err)
		return "", fmt.Errorf("failed to create Airwallex payment link: %w", err)
	}

	log.Printf("[Airwallex] Successfully created payment link: %s", link)
	return link, nil
}

// authenticate authenticates with Airwallex and returns a bearer token
func (a *AirwallexGenerator) authenticate() (string, error) {
	log.Printf("[Airwallex] Authenticating with client_id=%s, base_url=%s", a.clientID, a.baseURL)

	url := a.baseURL + "/api/v1/authentication/login"
	body := []byte(`{}`)

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create auth request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-client-id", a.clientID)
	req.Header.Set("x-api-key", a.apiKey)

	log.Printf("[Airwallex] Sending auth request to %s", url)
	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send auth request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read auth response: %w", err)
	}

	log.Printf("[Airwallex] Auth response status: %s", resp.Status)
	log.Printf("[Airwallex] Auth response body: %s", string(respBody))

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("authentication failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Token     string `json:"token"`
		ExpiresAt string `json:"expires_at"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse auth response: %w", err)
	}

	log.Printf("[Airwallex] Received token: %s, expires_at: %s", result.Token, result.ExpiresAt)
	return result.Token, nil
}

// createPaymentLink creates a payment link via Airwallex API
func (a *AirwallexGenerator) createPaymentLink(token string, data *PaymentLinkData) (string, error) {
	requestBody := a.buildPaymentLinkRequest(data)
	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	log.Printf("[Airwallex] Creating payment link with body: %s", string(bodyBytes))

	url := a.baseURL + "/api/v1/pa/payment_links/create"
	req, err := http.NewRequest("POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create payment link request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	log.Printf("[Airwallex] POST %s", url)
	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send payment link request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read payment link response: %w", err)
	}

	log.Printf("[Airwallex] Payment link response status: %s", resp.Status)
	log.Printf("[Airwallex] Payment link response body: %s", string(respBody))

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("payment link creation failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		PaymentLinkUrl string `json:"url"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse payment link response: %w", err)
	}

	if result.PaymentLinkUrl == "" {
		return "", fmt.Errorf("payment link URL not found in response")
	}

	return result.PaymentLinkUrl, nil
}

// buildPaymentLinkRequest constructs the request body for Airwallex payment link creation
func (a *AirwallexGenerator) buildPaymentLinkRequest(data *PaymentLinkData) map[string]interface{} {
	requestBody := map[string]interface{}{
		"amount":      data.Amount,
		"currency":    "USD",
		"title":       data.ServiceName,
		"description": data.ReferenceNumber,
		"reference":   fmt.Sprintf("slackbot-%d", time.Now().UnixNano()),
		"reusable":    false,
	}

	// Note: Airwallex may not support recurring payments in the same way as Stripe
	// For subscriptions, you might need to handle recurring billing differently
	if data.IsSubscription {
		log.Printf("[Airwallex] Warning: Subscription requested but may not be supported by Airwallex payment links")
		// You could add metadata or handle subscriptions through a different Airwallex API
		requestBody["metadata"] = map[string]interface{}{
			"is_subscription": true,
			"interval":        data.Interval,
			"interval_count":  data.IntervalCount,
		}
	}

	return requestBody
}
