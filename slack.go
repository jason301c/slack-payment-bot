package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/slack-go/slack"
)

// SlackService handles all Slack-related operations
type SlackService struct {
	client             *slack.Client
	signingSecret      string
	stripeGenerator    PaymentLinkGenerator
	airwallexGenerator PaymentLinkGenerator
}

var slackService *SlackService

// InitializeSlackService initializes the Slack service with configuration
func InitializeSlackService(config *Config) {
	slackService = &SlackService{
		client:        slack.New(config.SlackBotToken),
		signingSecret: config.SlackSigningSecret,
	}

	// Initialize payment generators
	slackService.stripeGenerator = NewStripeGenerator(config.StripeAPIKey)
	slackService.airwallexGenerator = NewAirwallexGenerator(config.AirwallexClientID, config.AirwallexAPIKey, config.AirwallexBaseURL)
}

// HandleSlackCommands processes incoming Slack slash command requests
func HandleSlackCommands(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received Slack command request: method=%s, url=%s, remote=%s", r.Method, r.URL.String(), r.RemoteAddr)

	// Verify request is from Slack
	verifier, err := slack.NewSecretsVerifier(r.Header, slackService.signingSecret)
	if err != nil {
		log.Printf("Error creating verifier: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Parse the Slack command
	r.Body = io.NopCloser(io.TeeReader(r.Body, &verifier))
	s, err := slack.SlashCommandParse(r)
	if err != nil {
		log.Printf("Error parsing slash command: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Authenticate the request
	if err = verifier.Ensure(); err != nil {
		log.Printf("Error verifying request: %v", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	log.Printf("Parsed Slack command: command=%s, text=%s, user_id=%s, channel_id=%s", s.Command, s.Text, s.UserID, s.ChannelID)

	// Determine provider and open modal
	var provider PaymentProvider
	switch s.Command {
	case "/create-stripe-link":
		provider = ProviderStripe
	case "/create-airwallex-link":
		provider = ProviderAirwallex
	default:
		respondToSlack(w, fmt.Sprintf("Unknown command: %s", s.Command))
		return
	}

	// If command text is provided, try to parse it directly
	if s.Text != "" {
		data, err := parseCommandArguments(s.Text)
		if err != nil {
			log.Printf("Error parsing command arguments: %v", err)
			respondToSlack(w, fmt.Sprintf("Invalid command format: %v\n\nUsage examples:\n• %s 19.99 \"Web Hosting\" INV-2024-001\n• %s 99.99 \"Consulting\" REF-ABC-123 true month 1", err, s.Command, s.Command))
			return
		}

		// Generate payment link
		var paymentLink string
		var generationErr error

		switch provider {
		case ProviderStripe:
			paymentLink, generationErr = slackService.stripeGenerator.GenerateLink(data)
		case ProviderAirwallex:
			paymentLink, generationErr = slackService.airwallexGenerator.GenerateLink(data)
		}

		if generationErr != nil {
			log.Printf("Error generating payment link: %v", generationErr)
			respondToSlack(w, fmt.Sprintf("Error generating payment link: %v", generationErr))
			return
		}

		// Send success message
		slackService.sendPaymentLinkMessage(s.UserID, s.ChannelID, data, paymentLink, provider)
		w.WriteHeader(http.StatusOK)
		return
	}

	// If no text provided, open modal
	if err := slackService.openPaymentLinkModal(s.TriggerID, provider); err != nil {
		log.Printf("Error opening modal: %v", err)
		respondToSlack(w, "Error opening payment form. Please try again.")
		return
	}

	// Respond with empty body for modal trigger
	w.WriteHeader(http.StatusOK)
}

// HandleSlackInteractions processes modal submissions and other interactions
func HandleSlackInteractions(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received Slack interaction request: method=%s, url=%s, remote=%s", r.Method, r.URL.String(), r.RemoteAddr)

	// Parse the interaction payload
	payload := r.FormValue("payload")
	var interaction slack.InteractionCallback
	if err := json.Unmarshal([]byte(payload), &interaction); err != nil {
		log.Printf("Error parsing interaction payload: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	switch interaction.Type {
	case slack.InteractionTypeViewSubmission:
		slackService.handleModalSubmission(w, &interaction)
	default:
		log.Printf("Unhandled interaction type: %s", interaction.Type)
		w.WriteHeader(http.StatusOK)
	}
}

// openPaymentLinkModal opens a modal for payment link creation
func (s *SlackService) openPaymentLinkModal(triggerID string, provider PaymentProvider) error {
	log.Printf("Opening payment link modal for provider: %s", provider)

	modalView := slack.ModalViewRequest{
		Type: slack.VTModal,
		Title: &slack.TextBlockObject{
			Type: slack.PlainTextType,
			Text: fmt.Sprintf("Create %s Payment Link", strings.Title(string(provider))),
		},
		Submit: &slack.TextBlockObject{
			Type: slack.PlainTextType,
			Text: "Create Link",
		},
		Close: &slack.TextBlockObject{
			Type: slack.PlainTextType,
			Text: "Cancel",
		},
		CallbackID: fmt.Sprintf("payment_link_modal_%s", provider),
		Blocks: slack.Blocks{
			BlockSet: []slack.Block{
				// Amount input
				&slack.InputBlock{
					Type:    slack.MBTInput,
					BlockID: "amount_block",
					Label: &slack.TextBlockObject{
						Type: slack.PlainTextType,
						Text: "Amount (USD)",
					},
					Element: &slack.PlainTextInputBlockElement{
						Type:     slack.METPlainTextInput,
						ActionID: "amount_input",
						Placeholder: &slack.TextBlockObject{
							Type: slack.PlainTextType,
							Text: "e.g., 19.99",
						},
					},
				},
				// Service name input
				&slack.InputBlock{
					Type:    slack.MBTInput,
					BlockID: "service_block",
					Label: &slack.TextBlockObject{
						Type: slack.PlainTextType,
						Text: "Service/Product Name",
					},
					Element: &slack.PlainTextInputBlockElement{
						Type:     slack.METPlainTextInput,
						ActionID: "service_input",
						Placeholder: &slack.TextBlockObject{
							Type: slack.PlainTextType,
							Text: "e.g., Web Hosting",
						},
					},
				},
				// Reference number input
				&slack.InputBlock{
					Type:     slack.MBTInput,
					BlockID:  "reference_block",
					Optional: true,
					Label: &slack.TextBlockObject{
						Type: slack.PlainTextType,
						Text: "Reference Number",
					},
					Element: &slack.PlainTextInputBlockElement{
						Type:     slack.METPlainTextInput,
						ActionID: "reference_input",
						Placeholder: &slack.TextBlockObject{
							Type: slack.PlainTextType,
							Text: "e.g., INV-2024-001",
						},
					},
				},
			},
		},
	}

	// Add subscription options for Stripe
	if provider == ProviderStripe {
		subscriptionBlocks := []slack.Block{
			// Subscription checkbox
			&slack.InputBlock{
				Type:     slack.MBTInput,
				BlockID:  "subscription_block",
				Optional: true,
				Label: &slack.TextBlockObject{
					Type: slack.PlainTextType,
					Text: "Subscription Options",
				},
				Element: &slack.CheckboxGroupsBlockElement{
					Type:     slack.METCheckboxGroups,
					ActionID: "subscription_checkbox",
					Options: []*slack.OptionBlockObject{
						{
							Text: &slack.TextBlockObject{
								Type: slack.PlainTextType,
								Text: "This is a recurring subscription",
							},
							Value: "is_subscription",
						},
					},
				},
			},
			// Billing interval
			&slack.InputBlock{
				Type:     slack.MBTInput,
				BlockID:  "interval_block",
				Optional: true,
				Label: &slack.TextBlockObject{
					Type: slack.PlainTextType,
					Text: "Billing Interval",
				},
				Element: &slack.SelectBlockElement{
					Type:     "static_select",
					ActionID: "interval_select",
					Placeholder: &slack.TextBlockObject{
						Type: slack.PlainTextType,
						Text: "Select billing period",
					},
					Options: []*slack.OptionBlockObject{
						{Text: &slack.TextBlockObject{Type: slack.PlainTextType, Text: "Monthly"}, Value: "month"},
						{Text: &slack.TextBlockObject{Type: slack.PlainTextType, Text: "Weekly"}, Value: "week"},
						{Text: &slack.TextBlockObject{Type: slack.PlainTextType, Text: "Yearly"}, Value: "year"},
					},
				},
			},
			// Interval count
			&slack.InputBlock{
				Type:     slack.MBTInput,
				BlockID:  "interval_count_block",
				Optional: true,
				Label: &slack.TextBlockObject{
					Type: slack.PlainTextType,
					Text: "Billing Frequency",
				},
				Element: &slack.SelectBlockElement{
					Type:     "static_select",
					ActionID: "interval_count_select",
					Placeholder: &slack.TextBlockObject{
						Type: slack.PlainTextType,
						Text: "Every X periods",
					},
					Options: []*slack.OptionBlockObject{
						{Text: &slack.TextBlockObject{Type: slack.PlainTextType, Text: "Every 1"}, Value: "1"},
						{Text: &slack.TextBlockObject{Type: slack.PlainTextType, Text: "Every 2"}, Value: "2"},
						{Text: &slack.TextBlockObject{Type: slack.PlainTextType, Text: "Every 3"}, Value: "3"},
						{Text: &slack.TextBlockObject{Type: slack.PlainTextType, Text: "Every 6"}, Value: "6"},
						{Text: &slack.TextBlockObject{Type: slack.PlainTextType, Text: "Every 12"}, Value: "12"},
					},
				},
			},
		}
		modalView.Blocks.BlockSet = append(modalView.Blocks.BlockSet, subscriptionBlocks...)
	}

	_, err := s.client.OpenView(triggerID, modalView)
	if err != nil {
		log.Printf("Error opening modal: %v", err)
		return fmt.Errorf("failed to open modal: %w", err)
	}

	return nil
}

// handleModalSubmission processes modal form submissions
func (s *SlackService) handleModalSubmission(w http.ResponseWriter, interaction *slack.InteractionCallback) {
	log.Printf("Handling modal submission for callback ID: %s", interaction.View.CallbackID)

	// Extract provider from callback ID
	callbackParts := strings.Split(interaction.View.CallbackID, "_")
	if len(callbackParts) < 3 {
		log.Printf("Invalid callback ID format: %s", interaction.View.CallbackID)
		respondWithError(w, "", "Invalid callback ID")
		return
	}
	provider := PaymentProvider(callbackParts[len(callbackParts)-1])

	// Parse form values
	values := interaction.View.State.Values

	// Extract and validate amount
	amountStr := values["amount_block"]["amount_input"].Value
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil || amount <= 0 {
		respondWithError(w, "amount_block", "Please enter a valid positive amount")
		return
	}

	// Extract and validate service name
	serviceName := values["service_block"]["service_input"].Value
	if serviceName == "" {
		respondWithError(w, "service_block", "Service name is required")
		return
	}

	// Extract reference number (optional)
	referenceNumber := fmt.Sprintf("REF-%d", time.Now().Unix())
	if refValue, ok := values["reference_block"]["reference_input"]; ok && refValue.Value != "" {
		referenceNumber = refValue.Value
	}

	// Parse subscription options (Stripe only)
	isSubscription := false
	interval := "month"
	intervalCount := int64(1)

	if provider == ProviderStripe {
		if subBlock, ok := values["subscription_block"]; ok {
			if subValue, ok := subBlock["subscription_checkbox"]; ok {
				isSubscription = len(subValue.SelectedOptions) > 0
			}
		}

		if isSubscription {
			if intervalBlock, ok := values["interval_block"]; ok {
				if intervalValue, ok := intervalBlock["interval_select"]; ok {
					if selectedOption := intervalValue.SelectedOption; selectedOption.Value != "" {
						interval = selectedOption.Value
					}
				}
			}

			if countBlock, ok := values["interval_count_block"]; ok {
				if countValue, ok := countBlock["interval_count_select"]; ok {
					if selectedOption := countValue.SelectedOption; selectedOption.Value != "" {
						if count, err := strconv.ParseInt(selectedOption.Value, 10, 64); err == nil {
							intervalCount = count
						}
					}
				}
			}
		}
	}

	// Create payment data
	paymentData := &PaymentLinkData{
		Amount:          amount,
		ServiceName:     serviceName,
		ReferenceNumber: referenceNumber,
		IsSubscription:  isSubscription,
		Interval:        interval,
		IntervalCount:   intervalCount,
	}

	// Generate payment link
	var paymentLink string
	var generationErr error

	switch provider {
	case ProviderStripe:
		paymentLink, generationErr = s.stripeGenerator.GenerateLink(paymentData)
	case ProviderAirwallex:
		paymentLink, generationErr = s.airwallexGenerator.GenerateLink(paymentData)
	default:
		generationErr = fmt.Errorf("unknown provider: %s", provider)
	}

	if generationErr != nil {
		log.Printf("Error generating %s payment link: %v", provider, generationErr)
		respondWithError(w, "", fmt.Sprintf("Error generating payment link: %v", generationErr))
		return
	}

	// Send success message
	channelID := interaction.Channel.ID
	s.sendPaymentLinkMessage(interaction.User.ID, channelID, paymentData, paymentLink, provider)

	// Close modal with success
	w.WriteHeader(http.StatusOK)
}

// sendPaymentLinkMessage sends the payment link to the Slack channel
func (s *SlackService) sendPaymentLinkMessage(userID, channelID string, data *PaymentLinkData, link string, provider PaymentProvider) {
	subscriptionText := ""
	if data.IsSubscription {
		subscriptionText = fmt.Sprintf("\nBilling: Every %d %s(s)", data.IntervalCount, data.Interval)
	}

	message := fmt.Sprintf("✅ *%s Payment Link Created*\n\n"+
		"💰 Amount: *$%.2f*%s\n"+
		"📦 Service: %s\n"+
		"🔢 Reference: `%s`\n\n"+
		"🔗 [Click here to pay](%s)",
		strings.Title(string(provider)),
		data.Amount,
		subscriptionText,
		data.ServiceName,
		data.ReferenceNumber,
		link)

	// Try to get channel from private metadata, fallback to DM
	if channelID == "" {
		// Send as DM to user
		if _, _, err := s.client.PostMessage(userID, slack.MsgOptionText(message, false)); err != nil {
			log.Printf("Error sending DM: %v", err)
		}
	} else {
		// Send to channel
		if _, _, err := s.client.PostMessage(channelID, slack.MsgOptionText(message, false)); err != nil {
			log.Printf("Error sending channel message: %v", err)
		}
	}
}

// Helper functions
func respondToSlack(w http.ResponseWriter, text string) {
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"text":          text,
		"response_type": "ephemeral",
	}
	json.NewEncoder(w).Encode(response)
}

func respondWithError(w http.ResponseWriter, blockID, message string) {
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"response_action": "errors",
		"errors": map[string]string{
			blockID: message,
		},
	}
	json.NewEncoder(w).Encode(response)
}
