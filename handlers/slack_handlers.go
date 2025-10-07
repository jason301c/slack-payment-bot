package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"paymentbot/models"
	"paymentbot/services"

	"github.com/slack-go/slack"
)

type SlackHandler struct {
	service *services.SlackService
}

func NewSlackHandler(svc *services.SlackService) *SlackHandler {
	return &SlackHandler{service: svc}
}

func (sh *SlackHandler) HandleSlackCommands(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received Slack command request: method=%s, url=%s, remote=%s", r.Method, r.URL.String(), r.RemoteAddr)
	verifier, err := slack.NewSecretsVerifier(r.Header, sh.service.GetSigningSecret())
	if err != nil {
		log.Printf("Error creating verifier: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	r.Body = io.NopCloser(io.TeeReader(r.Body, &verifier))
	sCmd, err := slack.SlashCommandParse(r)
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

	log.Printf("Parsed Slack command: command=%s, text=%s, user_id=%s, channel_id=%s, team_id=%s", sCmd.Command, sCmd.Text, sCmd.UserID, sCmd.ChannelID, sCmd.TeamID)

	var provider models.PaymentProvider
	switch sCmd.Command {
	case "/create-stripe-link":
		provider = models.ProviderStripe
	case "/create-airwallex-link":
		provider = models.ProviderAirwallex
	case "/create-invoice":
		// Handle invoice command separately
		if err := sh.service.OpenInvoiceModal(sCmd.TriggerID, sCmd.ChannelID, sCmd.TeamID); err != nil {
			log.Printf("Error opening invoice modal: %v", err)
			respondToSlack(w, "Error opening invoice form. Please try again.")
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	default:
		respondToSlack(w, fmt.Sprintf("Unknown command: %s", sCmd.Command))
		return
	}

	// Always open the modal, do not parse direct arguments
	if err := sh.service.OpenPaymentLinkModal(sCmd.TriggerID, provider, sCmd.ChannelID); err != nil {
		log.Printf("Error opening modal: %v", err)
		respondToSlack(w, "Error opening payment form. Please try again.")
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (sh *SlackHandler) HandleSlackInteractions(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received Slack interaction request: method=%s, url=%s, remote=%s", r.Method, r.URL.String(), r.RemoteAddr)
	payload := r.FormValue("payload")
	var interaction slack.InteractionCallback
	if err := json.Unmarshal([]byte(payload), &interaction); err != nil {
		log.Printf("Error parsing interaction payload: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	switch interaction.Type {
	case slack.InteractionTypeViewSubmission:
		if interaction.View.CallbackID == "invoice_modal" {
			sh.service.ProcessInvoiceSubmission(w, &interaction)
		} else {
			sh.service.ProcessModalSubmission(w, &interaction)
		}
	default:
		log.Printf("Unhandled interaction type: %s", interaction.Type)
		w.WriteHeader(http.StatusOK)
	}
}

func respondToSlack(w http.ResponseWriter, text string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"text": text})
}
