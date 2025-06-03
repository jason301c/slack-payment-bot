package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/slack-go/slack"
)

// Global Slack API client
var api *slack.Client

// Environment variables
var (
	slackBotToken      string
	slackSigningSecret string
	port               string
	stripeApiKey       string
	airwallexClientId  string
	airwallexApiKey    string
	airwallexBaseUrl   string
)

func init() {
	// Initialize environment variables
	slackBotToken = os.Getenv("SLACK_BOT_TOKEN")
	slackSigningSecret = os.Getenv("SLACK_SIGNING_SECRET")
	port = os.Getenv("PORT")
	stripeApiKey = os.Getenv("STRIPE_API_KEY")
	airwallexClientId = os.Getenv("AIRWALLEX_CLIENT_ID")
	airwallexApiKey = os.Getenv("AIRWALLEX_API_KEY")
	airwallexBaseUrl = os.Getenv("AIRWALLEX_BASE_URL")

	if slackBotToken == "" {
		log.Fatal("SLACK_BOT_TOKEN environment variable not set.")
	}
	if slackSigningSecret == "" {
		log.Fatal("SLACK_SIGNING_SECRET environment variable not set.")
	}
	if port == "" {
		port = "8080" // Default port
		log.Printf("PORT environment variable not set, defaulting to %s", port)
	}
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

	// Initialize the Slack API client
	api = slack.New(slackBotToken)
}

// PaymentLinkData holds the parsed data for creating a payment link.
type PaymentLinkData struct {
	Amount          float64
	ServiceName     string
	ReferenceNumber string
}

// parseCommandArguments parses the text from a Slack slash command.
// It expects the format: "[amount] [service_name] [reference_number]"
func parseCommandArguments(text string) (*PaymentLinkData, error) {
	parts := strings.Fields(text)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid arguments. Usage: [amount] [service_name] [reference_number]")
	}

	amountStr := parts[0]
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %v", err)
	}
	if amount <= 0 {
		return nil, fmt.Errorf("amount must be a positive number")
	}

	serviceName := parts[1]
	referenceNumber := parts[2]

	return &PaymentLinkData{
		Amount:          amount,
		ServiceName:     serviceName,
		ReferenceNumber: referenceNumber,
	}, nil
}

// handleSlackCommands processes incoming Slack slash command requests.
func handleSlackCommands(w http.ResponseWriter, r *http.Request) {
	verifier, err := slack.NewSecretsVerifier(r.Header, slackSigningSecret)
	if err != nil {
		log.Printf("Error creating verifier: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	r.Body = io.NopCloser(io.TeeReader(r.Body, &verifier))
	s, err := slack.SlashCommandParse(r)
	if err != nil {
		log.Printf("Error parsing slash command: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	if err = verifier.Ensure(); err != nil {
		log.Printf("Error verifying request: %v", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var responseText string
	var paymentLink string

	// Parse the command arguments
	linkData, err := parseCommandArguments(s.Text)
	if err != nil {
		responseText = fmt.Sprintf("Error: %s\nUsage: `/%s [amount] [service_name] [reference_number]`", err.Error(), s.Command[1:])
	} else {
		switch s.Command {
		case "/create-airwallex-link":
			paymentLink = GenerateAirwallexLink(linkData)
			responseText = fmt.Sprintf("Airwallex Payment Link for *%s*\nAmount: *$%.2f*\nReference: `%s`\nLink: <%s|Click here to pay>",
				linkData.ServiceName, linkData.Amount, linkData.ReferenceNumber, paymentLink)
		case "/create-stripe-link":
			paymentLink = GenerateStripeLink(linkData)
			responseText = fmt.Sprintf("Stripe Payment Link for *%s*\nAmount: *$%.2f*\nReference: `%s`\nLink: <%s|Click here to pay>",
				linkData.ServiceName, linkData.Amount, linkData.ReferenceNumber, paymentLink)
		default:
			responseText = fmt.Sprintf("Unknown command: %s", s.Command)
		}
	}

	// Send an immediate response to Slack
	w.Header().Set("Content-Type", "application/json")
	type slackResponse struct {
		Text         string `json:"text"`
		ResponseType string `json:"response_type"`
	}
	resp := slackResponse{
		Text:         responseText,
		ResponseType: "in_channel", // or "ephemeral"
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func main() {
	log.Printf("Starting Slack bot server on :%s", port)
	http.HandleFunc("/slack/commands", handleSlackCommands)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
