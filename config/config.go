package config

import (
	"log"
	"os"
)

// Config holds application configuration
type Config struct {
	SlackBotToken      string
	SlackSigningSecret string
	Port               string
	StripeAPIKey       string
	AirwallexClientID  string
	AirwallexAPIKey    string
	AirwallexBaseURL   string
}

func LoadConfig() *Config {
	cfg := &Config{
		SlackBotToken:      os.Getenv("SLACK_BOT_TOKEN"),
		SlackSigningSecret: os.Getenv("SLACK_SIGNING_SECRET"),
		Port:               os.Getenv("PORT"),
		StripeAPIKey:       os.Getenv("STRIPE_API_KEY"),
		AirwallexClientID:  os.Getenv("AIRWALLEX_CLIENT_ID"),
		AirwallexAPIKey:    os.Getenv("AIRWALLEX_API_KEY"),
		AirwallexBaseURL:   os.Getenv("AIRWALLEX_BASE_URL"),
	}

	if cfg.SlackBotToken == "" {
		log.Fatal("SLACK_BOT_TOKEN environment variable not set.")
	}
	if cfg.SlackSigningSecret == "" {
		log.Fatal("SLACK_SIGNING_SECRET environment variable not set.")
	}
	if cfg.Port == "" {
		cfg.Port = "8080"
		log.Printf("PORT environment variable not set, defaulting to %s", cfg.Port)
	}
	if cfg.StripeAPIKey == "" {
		log.Fatal("STRIPE_API_KEY environment variable not set.")
	}
	if cfg.AirwallexClientID == "" {
		log.Fatal("AIRWALLEX_CLIENT_ID environment variable not set.")
	}
	if cfg.AirwallexAPIKey == "" {
		log.Fatal("AIRWALLEX_API_KEY environment variable not set.")
	}
	if cfg.AirwallexBaseURL == "" {
		cfg.AirwallexBaseURL = "https://api.airwallex.com"
	}

	return cfg
}
