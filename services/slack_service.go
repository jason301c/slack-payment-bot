package services

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"paymentbot/config"
	"paymentbot/models"
	"paymentbot/payment"

	"github.com/slack-go/slack"
)

type SlackService struct {
	client             *slack.Client
	signingSecret      string
	stripeGenerator    payment.PaymentLinkGenerator
	airwallexGenerator payment.PaymentLinkGenerator
}

func NewSlackService(cfg *config.Config, stripeGen payment.PaymentLinkGenerator, airwallexGen payment.PaymentLinkGenerator) *SlackService {
	return &SlackService{
		client:             slack.New(cfg.SlackBotToken),
		signingSecret:      cfg.SlackSigningSecret,
		stripeGenerator:    stripeGen,
		airwallexGenerator: airwallexGen,
	}
}

func (s *SlackService) GetSigningSecret() string {
	return s.signingSecret
}

func (s *SlackService) OpenPaymentLinkModal(triggerID string, provider models.PaymentProvider, channelID string) error {
	log.Printf("Opening payment link modal for provider: %s, channel: %s", provider, channelID)
	modalView := BuildPaymentModalView(provider, channelID)

	_, err := s.client.OpenView(triggerID, modalView)
	if err != nil {
		log.Printf("Error opening modal: %v", err)
		return fmt.Errorf("failed to open modal: %w", err)
	}
	return nil
}

func (s *SlackService) GenerateLinkForProvider(data *models.PaymentLinkData, provider models.PaymentProvider) (string, string, error) {
	var paymentLink, paymentID string
	var generationErr error

	switch provider {
	case models.ProviderStripe:
		paymentLink, paymentID, generationErr = s.stripeGenerator.GenerateLink(data)
	case models.ProviderAirwallex:
		paymentLink, paymentID, generationErr = s.airwallexGenerator.GenerateLink(data)
	default:
		return "", "", fmt.Errorf("unknown provider: %s", provider)
	}
	return paymentLink, paymentID, generationErr
}

func (s *SlackService) SendPaymentLinkMessage(userID, channelID string, data *models.PaymentLinkData, link, paymentID string, provider models.PaymentProvider) {
	msg := fmt.Sprintf(
		"<@%s> Here is your %s payment link for *%s* (Amount: $%.2f):\n%s",
		userID, strings.Title(string(provider)), data.ServiceName, data.Amount, link,
	)
	if paymentID != "" {
		msg += fmt.Sprintf("\nPayment ID: `%s`", paymentID)
	}
	if data.IsSubscription && data.EndDateCycles > 0 {
		msg += fmt.Sprintf("\nEnd Date: %d cycles (%d %s payments)", data.EndDateCycles, data.EndDateCycles, data.Interval)
	}
	_, _, err := s.client.PostMessage(channelID, slack.MsgOptionText(msg, false))
	if err != nil {
		log.Printf("Error sending payment link message to channel %s: %v", channelID, err)
		// Fallback: send to user's DM with debug note
		debugMsg := msg + fmt.Sprintf("\n\n:warning: _This message was not sent to the channel because of: %v. Perhaps add the bot to the channel?_", err)
		_, _, dmErr := s.client.PostMessage(userID, slack.MsgOptionText(debugMsg, false))
		if dmErr != nil {
			log.Printf("Error sending fallback DM to user %s: %v", userID, dmErr)
		}
	}
}

func (s *SlackService) ProcessModalSubmission(w http.ResponseWriter, interaction *slack.InteractionCallback) {
	log.Printf("Handling modal submission for callback ID: %s", interaction.View.CallbackID)

	// Extract provider from callback ID
	callbackParts := strings.Split(interaction.View.CallbackID, "_")
	provider := models.PaymentProvider(callbackParts[len(callbackParts)-1])

	values := interaction.View.State.Values
	amountStr := values["amount_block"]["amount_input"].Value
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil || amount <= 0 {
		respondWithError(w, "amount_block", "Please enter a valid positive amount")
		return
	}
	serviceName := values["service_block"]["service_input"].Value
	if serviceName == "" {
		respondWithError(w, "service_block", "Service name cannot be empty")
		return
	}
	referenceNumber := values["reference_block"]["reference_input"].Value
	if referenceNumber == "" {
		referenceNumber = fmt.Sprintf("REF-%d", time.Now().Unix())
	}

	isSubscription := false
	interval := "month"
	intervalCount := int64(1)
	endDateCycles := int64(0)

	if provider == models.ProviderStripe {
		// Check for subscription checkbox
		if subBlock, ok := values["subscription_block"]; ok {
			if subElem, ok := subBlock["subscription_checkbox"]; ok && len(subElem.SelectedOptions) > 0 {
				isSubscription = true
			}
		}
		// Interval select
		if intervalBlock, ok := values["interval_block"]; ok {
			if intervalElem, ok := intervalBlock["interval_select"]; ok && intervalElem.SelectedOption.Value != "" {
				interval = intervalElem.SelectedOption.Value
			}
		}
		// Interval count select
		if countBlock, ok := values["interval_count_block"]; ok {
			if countElem, ok := countBlock["interval_count_select"]; ok && countElem.SelectedOption.Value != "" {
				parsed, err := strconv.ParseInt(countElem.SelectedOption.Value, 10, 64)
				if err == nil && parsed > 0 {
					intervalCount = parsed
				}
			}
		}
		// End date cycles input
		if endDateBlock, ok := values["end_date_block"]; ok {
			if endDateElem, ok := endDateBlock["end_date_input"]; ok && endDateElem.Value != "" {
				parsed, err := strconv.ParseInt(strings.TrimSpace(endDateElem.Value), 10, 64)
				if err != nil {
					respondWithError(w, "end_date_block", "Please enter a valid number for end date cycles")
					return
				}
				if parsed <= 0 {
					respondWithError(w, "end_date_block", "End date cycles must be a positive number")
					return
				}
				endDateCycles = parsed
			}
		}
	}

	internalReference := ""
	if provider == models.ProviderAirwallex {
		internalReference = values["internal_reference_block"]["internal_reference_input"].Value
	}

	paymentData := &models.PaymentLinkData{
		Amount:            amount,
		ServiceName:       serviceName,
		ReferenceNumber:   referenceNumber,
		IsSubscription:    isSubscription,
		Interval:          interval,
		IntervalCount:     intervalCount,
		EndDateCycles:     endDateCycles,
		InternalReference: internalReference,
	}

	paymentLink, paymentID, generationErr := s.GenerateLinkForProvider(paymentData, provider)
	if generationErr != nil {
		log.Printf("Error generating %s payment link: %v", provider, generationErr)
		respondWithError(w, "", fmt.Sprintf("Error generating payment link: %v", generationErr))
		return
	}

	channelID := interaction.Channel.ID
	if channelID == "" {
		// Try to get channel from private metadata
		if interaction.View.PrivateMetadata != "" {
			channelID = interaction.View.PrivateMetadata
		} else {
			// Fallback to DM the user if no channel context is available
			channelID = interaction.User.ID
		}
	}

	log.Printf("Sending payment link message to user: %s, channel: %s, payment link: %s, payment ID: %s, provider: %s", interaction.User.ID, channelID, paymentLink, paymentID, provider)
	s.SendPaymentLinkMessage(interaction.User.ID, channelID, paymentData, paymentLink, paymentID, provider)
	w.WriteHeader(http.StatusOK)
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
