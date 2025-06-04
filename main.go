package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

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
		airwallexBaseUrl = "https://api.airwallex.com" // Default to prod url
	}

	// Initialize the Slack API client
	api = slack.New(slackBotToken)
}

// Centralized data structure for storing input and generating output
type PaymentLinkData struct {
	Amount          float64
	ServiceName     string
	ReferenceNumber string
}

// handleSlackCommands processes incoming Slack slash command requests.
func handleSlackCommands(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received Slack command request: method=%s, url=%s, remote=%s", r.Method, r.URL.String(), r.RemoteAddr)

	// Step 1: Verify request is from Slack
	verifier, err := slack.NewSecretsVerifier(r.Header, slackSigningSecret)
	if err != nil {
		log.Printf("Error creating verifier: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Step 2: Parse the Slack command
	r.Body = io.NopCloser(io.TeeReader(r.Body, &verifier))
	s, err := slack.SlashCommandParse(r)
	if err != nil {
		log.Printf("Error parsing slash command: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	log.Printf("Parsed Slack command: command=%s, text=%s, user_id=%s, channel_id=%s", s.Command, s.Text, s.UserID, s.ChannelID)

	// Step 3: Authenticate the request
	if err = verifier.Ensure(); err != nil {
		log.Printf("Error verifying request: %v", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Step 4: Parse command arguments
	linkData, err := parseCommandArguments(s.Text)
	if err != nil {
		usage := "Usage: /" + s.Command[1:] + " [amount] [service_name] [reference_number]"
		example := "Example: /" + s.Command[1:] + " 19.99 \"Web Hosting\" 2024-INV-001"
		errMsg := "*Error parsing arguments: " + err.Error() + "*\n" + usage + "\n" + example + "\n" + "Please try again."
		respondToSlack(w, errMsg)
		return
	}
	log.Printf("Parsed arguments: amount=%.2f, service_name=%s, reference_number=%s", linkData.Amount, linkData.ServiceName, linkData.ReferenceNumber)

	// Step 5: Generate payment link based on command
	var responseText string
	switch s.Command {
	case "/create-airwallex-link":
		log.Printf("Generating Airwallex link for: %+v", linkData)
		paymentLink := GenerateAirwallexLink(linkData)
		log.Printf("Airwallex link result: %s", paymentLink)
		responseText = fmt.Sprintf("Airwallex Payment Link for *%s*\nAmount: *$%.2f*\nReference: `%s`\nLink: <%s|Click here to pay>",
			linkData.ServiceName, linkData.Amount, linkData.ReferenceNumber, paymentLink)
	case "/create-stripe-link":
		log.Printf("Generating Stripe link for: %+v", linkData)
		paymentLink := GenerateStripeLink(linkData)
		log.Printf("Stripe link result: %s", paymentLink)
		responseText = fmt.Sprintf("Stripe Payment Link for *%s*\nAmount: *$%.2f*\nReference: `%s`\nLink: <%s|Click here to pay>",
			linkData.ServiceName, linkData.Amount, linkData.ReferenceNumber, paymentLink)
	default:
		log.Printf("Unknown command received: %s", s.Command)
		responseText = fmt.Sprintf("Unknown command: %s", s.Command)
	}

	respondToSlack(w, responseText)
}

// respondToSlack sends a JSON response to Slack in the expected format.
func respondToSlack(w http.ResponseWriter, text string) {
	w.Header().Set("Content-Type", "application/json")
	type slackResponse struct {
		Text         string `json:"text"`
		ResponseType string `json:"response_type"`
	}
	resp := slackResponse{
		Text:         text,
		ResponseType: "in_channel", // or "ephemeral"
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// Main function to start the server
func main() {
	log.Printf("Starting Slack bot server on :%s", port)
	http.HandleFunc("/slack/commands", handleSlackCommands)
	log.Printf("Registered /slack/commands handler. Ready to receive requests.")
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
