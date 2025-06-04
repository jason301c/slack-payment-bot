package main

import (
	"log"
	"net/http"
	"os"
)

// Global configuration
var config *Config

func init() {
	// Initialize configuration from environment variables
	config = &Config{
		SlackBotToken:      os.Getenv("SLACK_BOT_TOKEN"),
		SlackSigningSecret: os.Getenv("SLACK_SIGNING_SECRET"),
		Port:               os.Getenv("PORT"),
		StripeAPIKey:       os.Getenv("STRIPE_API_KEY"),
		AirwallexClientID:  os.Getenv("AIRWALLEX_CLIENT_ID"),
		AirwallexAPIKey:    os.Getenv("AIRWALLEX_API_KEY"),
		AirwallexBaseURL:   os.Getenv("AIRWALLEX_BASE_URL"),
	}

	// Validate required environment variables
	if config.SlackBotToken == "" {
		log.Fatal("SLACK_BOT_TOKEN environment variable not set.")
	}
	if config.SlackSigningSecret == "" {
		log.Fatal("SLACK_SIGNING_SECRET environment variable not set.")
	}
	if config.Port == "" {
		config.Port = "8080" // Default port
		log.Printf("PORT environment variable not set, defaulting to %s", config.Port)
	}
	if config.StripeAPIKey == "" {
		log.Fatal("STRIPE_API_KEY environment variable not set.")
	}
	if config.AirwallexClientID == "" {
		log.Fatal("AIRWALLEX_CLIENT_ID environment variable not set.")
	}
	if config.AirwallexAPIKey == "" {
		log.Fatal("AIRWALLEX_API_KEY environment variable not set.")
	}
	if config.AirwallexBaseURL == "" {
		config.AirwallexBaseURL = "https://api.airwallex.com" // Default to prod url
	}

	// Initialize services
	InitializeSlackService(config)
}

// Main function to start the server
func main() {
	log.Printf("Starting Slack bot server on :%s", config.Port)

	// Register handlers
	http.HandleFunc("/slack/commands", HandleSlackCommands)
	http.HandleFunc("/slack/interactions", HandleSlackInteractions)

	log.Printf("Registered handlers. Ready to receive requests.")
	log.Fatal(http.ListenAndServe(":"+config.Port, nil))
}
