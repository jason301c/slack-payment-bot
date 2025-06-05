package main

import (
	"log"
	"net/http"

	"paymentbot/config"
	"paymentbot/handlers"
	"paymentbot/payment"
	"paymentbot/services"
)

func main() {
	appConfig := config.LoadConfig()
	log.Printf("Starting Slack bot server on :%s", appConfig.Port)

	// Debug: Print the first 8 chars of each token (never print full tokens)
	log.Printf("Slack Bot Token: %s...", appConfig.SlackBotToken[:8])
	log.Printf("Slack Signing Secret: %s...", appConfig.SlackSigningSecret[:8])
	log.Printf("Stripe API Key: %s...", appConfig.StripeAPIKey[:8])
	log.Printf("Airwallex Client ID: %s...", appConfig.AirwallexClientID[:8])
	log.Printf("Airwallex API Key: %s...", appConfig.AirwallexAPIKey[:8])

	// Initialize Payment Generators
	stripeGenerator := payment.NewStripeGenerator(appConfig.StripeAPIKey)
	airwallexGenerator := payment.NewAirwallexGenerator(
		appConfig.AirwallexClientID,
		appConfig.AirwallexAPIKey,
		appConfig.AirwallexBaseURL,
	)

	// Initialize Slack Service
	slackService := services.NewSlackService(appConfig, stripeGenerator, airwallexGenerator)

	// Initialize Slack Handler
	slackHandler := handlers.NewSlackHandler(slackService)

	// Register handlers
	http.HandleFunc("/slack/commands", slackHandler.HandleSlackCommands)
	http.HandleFunc("/slack/interactions", slackHandler.HandleSlackInteractions)

	log.Printf("Registered handlers. Ready to receive requests.")
	log.Fatal(http.ListenAndServe(":"+appConfig.Port, nil))
}
