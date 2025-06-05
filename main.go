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
