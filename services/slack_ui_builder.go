package services

import (
	"fmt"
	"strings"

	"paymentbot/models"

	"github.com/slack-go/slack"
)

func newPlainTextBlock(text string) *slack.TextBlockObject {
	return slack.NewTextBlockObject(slack.PlainTextType, text, false, false)
}

func BuildPaymentModalView(provider models.PaymentProvider) slack.ModalViewRequest {
	modalTitle := newPlainTextBlock(fmt.Sprintf("%s Payment", strings.Title(string(provider))))
	submitText := newPlainTextBlock("Create Link")
	closeText := newPlainTextBlock("Cancel")

	amountLabel := newPlainTextBlock("Amount (USD)")
	amountPlaceholder := newPlainTextBlock("e.g., 19.99")
	amountElement := slack.NewPlainTextInputBlockElement(amountPlaceholder, "amount_input")
	amountBlock := slack.NewInputBlock("amount_block", amountLabel, nil, amountElement)
	amountBlock.Optional = false

	serviceLabel := newPlainTextBlock("Service/Product Name")
	servicePlaceholder := newPlainTextBlock("e.g., Web Hosting")
	serviceElement := slack.NewPlainTextInputBlockElement(servicePlaceholder, "service_input")
	serviceBlock := slack.NewInputBlock("service_block", serviceLabel, nil, serviceElement)
	serviceBlock.Optional = false

	referenceLabel := newPlainTextBlock("Description")
	referencePlaceholder := newPlainTextBlock("Enter your description here")
	referenceHint := newPlainTextBlock("Appears at checkout.")
	referenceElement := slack.NewPlainTextInputBlockElement(referencePlaceholder, "reference_input")
	referenceBlock := slack.NewInputBlock("reference_block", referenceLabel, referenceHint, referenceElement)
	referenceBlock.Optional = true

	allBlocks := []slack.Block{amountBlock, serviceBlock, referenceBlock}

	if provider == models.ProviderStripe {
		subscriptionLabel := newPlainTextBlock("Subscription Options")
		subOptionText := newPlainTextBlock("This is a recurring subscription")
		subOption := slack.NewOptionBlockObject("is_subscription", subOptionText, nil)
		subscriptionElement := slack.NewCheckboxGroupsBlockElement("subscription_checkbox", subOption)
		subscriptionBlock := slack.NewInputBlock("subscription_block", subscriptionLabel, nil, subscriptionElement)
		subscriptionBlock.Optional = true

		intervalLabel := newPlainTextBlock("Billing Interval")
		intervalPlaceholder := newPlainTextBlock("Select billing period")
		monthOption := slack.NewOptionBlockObject("month", newPlainTextBlock("Monthly"), nil)
		weekOption := slack.NewOptionBlockObject("week", newPlainTextBlock("Weekly"), nil)
		yearOption := slack.NewOptionBlockObject("year", newPlainTextBlock("Yearly"), nil)
		intervalElement := slack.NewOptionsSelectBlockElement(slack.OptTypeStatic, intervalPlaceholder, "interval_select", monthOption, weekOption, yearOption)
		intervalElement.InitialOption = monthOption
		intervalBlock := slack.NewInputBlock("interval_block", intervalLabel, nil, intervalElement)
		intervalBlock.Optional = true

		countLabel := newPlainTextBlock("Billing Frequency")
		countPlaceholder := newPlainTextBlock("Every X periods")
		countOpts := []*slack.OptionBlockObject{
			slack.NewOptionBlockObject("1", newPlainTextBlock("Every 1"), nil),
			slack.NewOptionBlockObject("2", newPlainTextBlock("Every 2"), nil),
			slack.NewOptionBlockObject("3", newPlainTextBlock("Every 3"), nil),
			slack.NewOptionBlockObject("6", newPlainTextBlock("Every 6"), nil),
			slack.NewOptionBlockObject("12", newPlainTextBlock("Every 12"), nil),
		}
		countElement := slack.NewOptionsSelectBlockElement(slack.OptTypeStatic, countPlaceholder, "interval_count_select", countOpts...)
		countElement.InitialOption = countOpts[0]
		countBlock := slack.NewInputBlock("interval_count_block", countLabel, nil, countElement)
		countBlock.Optional = true

		allBlocks = append(allBlocks, subscriptionBlock, intervalBlock, countBlock)
	}

	if provider == models.ProviderAirwallex {
		internalRefLabel := newPlainTextBlock("Internal reference")
		internalRefPlaceholder := newPlainTextBlock("e.g. REF-123")
		internalRefHint := newPlainTextBlock("This reference is only visible to your account. It provides information about this transaction for your records.")
		internalRefElement := slack.NewPlainTextInputBlockElement(internalRefPlaceholder, "internal_reference_input")
		internalRefBlock := slack.NewInputBlock("internal_reference_block", internalRefLabel, internalRefHint, internalRefElement)
		internalRefBlock.Optional = true
		allBlocks = append(allBlocks, internalRefBlock)
	}

	return slack.ModalViewRequest{
		Type:          slack.VTModal,
		Title:         modalTitle,
		Submit:        submitText,
		Close:         closeText,
		CallbackID:    fmt.Sprintf("payment_link_modal_%s", provider),
		ClearOnClose:  true,
		NotifyOnClose: false,
		Blocks:        slack.Blocks{BlockSet: allBlocks},
	}
}
