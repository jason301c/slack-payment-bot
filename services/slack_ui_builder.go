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

func BuildPaymentModalView(provider models.PaymentProvider, privateMetadata string) slack.ModalViewRequest {
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

		endDateLabel := newPlainTextBlock("End Date (optional)")
		endDatePlaceholder := newPlainTextBlock("Enter number of cycles (e.g., 6)")
		endDateHint := newPlainTextBlock("Leave empty for no end date. Enter a number to limit subscription to that many billing cycles.")
		endDateElement := slack.NewPlainTextInputBlockElement(endDatePlaceholder, "end_date_input")
		endDateBlock := slack.NewInputBlock("end_date_block", endDateLabel, endDateHint, endDateElement)
		endDateBlock.Optional = true

		allBlocks = append(allBlocks, subscriptionBlock, intervalBlock, countBlock, endDateBlock)
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
		Type:            slack.VTModal,
		Title:           modalTitle,
		Submit:          submitText,
		Close:           closeText,
		CallbackID:      fmt.Sprintf("payment_link_modal_%s", provider),
		ClearOnClose:    true,
		NotifyOnClose:   false,
		Blocks:          slack.Blocks{BlockSet: allBlocks},
		PrivateMetadata: privateMetadata,
	}
}

func BuildInvoiceModalView(privateMetadata string) slack.ModalViewRequest {
	modalTitle := newPlainTextBlock("Create Invoice")
	submitText := newPlainTextBlock("Generate Invoice")
	closeText := newPlainTextBlock("Cancel")

	// Basic invoice fields
	invoiceNumberLabel := newPlainTextBlock("Invoice Number")
	invoiceNumberPlaceholder := newPlainTextBlock("e.g., 935")
	invoiceNumberElement := slack.NewPlainTextInputBlockElement(invoiceNumberPlaceholder, "invoice_number_input")
	invoiceNumberBlock := slack.NewInputBlock("invoice_number_block", invoiceNumberLabel, nil, invoiceNumberElement)
	invoiceNumberBlock.Optional = false

	clientNameLabel := newPlainTextBlock("Client Name")
	clientNamePlaceholder := newPlainTextBlock("e.g., Acme Corporation")
	clientNameElement := slack.NewPlainTextInputBlockElement(clientNamePlaceholder, "client_name_input")
	clientNameBlock := slack.NewInputBlock("client_name_block", clientNameLabel, nil, clientNameElement)
	clientNameBlock.Optional = false

	clientAddressLabel := newPlainTextBlock("Client Address (Optional)")
	clientAddressPlaceholder := newPlainTextBlock("123 Main St, City, State 12345")
	clientAddressElement := slack.NewPlainTextInputBlockElement(clientAddressPlaceholder, "client_address_input")
	clientAddressBlock := slack.NewInputBlock("client_address_block", clientAddressLabel, nil, clientAddressElement)
	clientAddressBlock.Optional = true

	clientEmailLabel := newPlainTextBlock("Client Email")
	clientEmailPlaceholder := newPlainTextBlock("client@example.com")
	clientEmailElement := slack.NewPlainTextInputBlockElement(clientEmailPlaceholder, "client_email_input")
	clientEmailBlock := slack.NewInputBlock("client_email_block", clientEmailLabel, nil, clientEmailElement)
	clientEmailBlock.Optional = false

	dateDueLabel := newPlainTextBlock("Due Date")
	dateDuePlaceholder := newPlainTextBlock("e.g., 2024-12-31")
	dateDueElement := slack.NewPlainTextInputBlockElement(dateDuePlaceholder, "date_due_input")
	dateDueBlock := slack.NewInputBlock("date_due_block", dateDueLabel, nil, dateDueElement)
	dateDueBlock.Optional = false

	// Line items section header
	lineItemsHeader := slack.NewSectionBlock(
		newPlainTextBlock("Invoice Line Items"),
		nil,
		nil,
	)

	allBlocks := []slack.Block{
		invoiceNumberBlock,
		clientNameBlock,
		clientAddressBlock,
		clientEmailBlock,
		dateDueBlock,
		lineItemsHeader,
		slack.NewDividerBlock(),
	}

	// Add 5 line items by default (can be expanded)
	for i := 0; i < 5; i++ {
		serviceLabel := newPlainTextBlock(fmt.Sprintf("Service Description %d", i+1))
		servicePlaceholder := newPlainTextBlock("e.g., Web Development Services")
		serviceElement := slack.NewPlainTextInputBlockElement(servicePlaceholder, fmt.Sprintf("service_input_%d", i))
		serviceBlock := slack.NewInputBlock(fmt.Sprintf("service_%d", i), serviceLabel, nil, serviceElement)
		serviceBlock.Optional = (i > 0) // First item is required

		unitPriceLabel := newPlainTextBlock(fmt.Sprintf("Unit Price %d ($)", i+1))
		unitPricePlaceholder := newPlainTextBlock("e.g., 150.00")
		unitPriceElement := slack.NewPlainTextInputBlockElement(unitPricePlaceholder, fmt.Sprintf("unit_price_input_%d", i))
		unitPriceBlock := slack.NewInputBlock(fmt.Sprintf("unit_price_%d", i), unitPriceLabel, nil, unitPriceElement)
		unitPriceBlock.Optional = (i > 0) // First item is required

		quantityLabel := newPlainTextBlock(fmt.Sprintf("Quantity %d", i+1))
		quantityPlaceholder := newPlainTextBlock("e.g., 1")
		quantityElement := slack.NewPlainTextInputBlockElement(quantityPlaceholder, fmt.Sprintf("quantity_input_%d", i))
		quantityBlock := slack.NewInputBlock(fmt.Sprintf("quantity_%d", i), quantityLabel, nil, quantityElement)
		quantityBlock.Optional = (i > 0) // First item is required

		allBlocks = append(allBlocks, serviceBlock, unitPriceBlock, quantityBlock)

		// Add divider between items (except after last one)
		if i < 4 {
			allBlocks = append(allBlocks, slack.NewDividerBlock())
		}
	}

	return slack.ModalViewRequest{
		Type:            slack.VTModal,
		Title:           modalTitle,
		Submit:          submitText,
		Close:           closeText,
		CallbackID:      "invoice_modal",
		ClearOnClose:    true,
		NotifyOnClose:   false,
		Blocks:          slack.Blocks{BlockSet: allBlocks},
		PrivateMetadata: privateMetadata,
	}
}
