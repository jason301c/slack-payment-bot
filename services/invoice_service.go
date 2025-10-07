package services

import (
	"bytes"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"paymentbot/models"

	"github.com/jung-kurt/gofpdf"
	"github.com/slack-go/slack"
)

type InvoiceService struct {
	slackClient *slack.Client
}

func NewInvoiceService(slackClient *slack.Client) *InvoiceService {
	return &InvoiceService{
		slackClient: slackClient,
	}
}

func getCurrencySymbol(currency string) string {
	symbols := map[string]string{
		"USD": "$",
		"EUR": "€",
		"GBP": "£",
		"JPY": "¥",
		"HKD": "HK$",
		"CAD": "C$",
		"AUD": "A$",
	}
	if symbol, exists := symbols[currency]; exists {
		return symbol
	}
	return "$" // Default to USD symbol
}

func (is *InvoiceService) GenerateInvoicePDF(invoice *models.InvoiceData) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	// Set font
	pdf.SetFont("Arial", "", 10)

	// Company Information (left side)
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(0, 8, "ZEFI ECOMMERCE LIMITED")
	pdf.Ln(6)

	pdf.SetFont("Arial", "", 9)
	pdf.Cell(0, 5, "Glenealy Central")
	pdf.Ln(4)
	pdf.Cell(0, 5, "Unit 2A, 17/F, Glenealy Tower, No.1 Hong Kong")
	pdf.Ln(4)
	pdf.Cell(0, 5, "+61 466 598 489")
	pdf.Ln(15)

	// Invoice title and number (right side)
	pdf.SetFont("Arial", "B", 24)
	pdf.Cell(0, 10, "INVOICE")
	pdf.Ln(15)

	// Invoice details
	pdf.SetFont("Arial", "", 10)
	pdf.Cell(60, 6, fmt.Sprintf("Invoice Number: %s", invoice.InvoiceNumber))
	pdf.Cell(60, 6, fmt.Sprintf("Date: %s", time.Now().Format("January 2, 2006")))
	pdf.Ln(6)
	pdf.Cell(60, 6, fmt.Sprintf("Due Date: %s", invoice.DateDue))
	pdf.Cell(60, 6, fmt.Sprintf("Currency: %s", invoice.Currency))
	pdf.Ln(15)

	// Bill To section
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(0, 8, "Bill To:")
	pdf.Ln(6)

	pdf.SetFont("Arial", "", 10)
	pdf.Cell(0, 5, invoice.ClientName)
	pdf.Ln(5)
	if invoice.ClientAddress != "" {
		pdf.Cell(0, 5, invoice.ClientAddress)
		pdf.Ln(5)
	}
	if invoice.ClientEmail != "" {
		pdf.Cell(0, 5, invoice.ClientEmail)
		pdf.Ln(15)
	} else {
		pdf.Ln(10)
	}

	// Table headers
	pdf.SetFont("Arial", "B", 11)
	pdf.SetFillColor(240, 240, 240)
	pdf.Cell(100, 8, "Description")
	pdf.Cell(25, 8, "Qty")
	pdf.Cell(35, 8, "Unit Price")
	pdf.Cell(40, 8, "Amount")
	pdf.Ln(10)

	// Table line
	pdf.SetDrawColor(200, 200, 200)
	pdf.Line(10, pdf.GetY(), 200, pdf.GetY())
	pdf.Ln(5)

	// Line items
	pdf.SetFont("Arial", "", 10)
	var subtotal float64
	for i, item := range invoice.LineItems {
		// Description
		pdf.Cell(100, 6, item.ServiceDescription)

		// Quantity
		quantity := fmt.Sprintf("%d", item.Quantity)
		pdf.Cell(25, 6, quantity)

		// Unit Price
		currencySymbol := getCurrencySymbol(invoice.Currency)
		unitPriceStr := fmt.Sprintf("%s%.2f", currencySymbol, item.UnitPrice)
		pdf.Cell(35, 6, unitPriceStr)

		// Amount (qty * unit price)
		lineTotal := float64(item.Quantity) * item.UnitPrice
		amountStr := fmt.Sprintf("%s%.2f", currencySymbol, lineTotal)
		pdf.Cell(40, 6, amountStr)
		pdf.Ln(6)

		subtotal += lineTotal

		// Add spacing between items
		if i < len(invoice.LineItems)-1 {
			pdf.Ln(2)
		}
	}

	// Totals section
	pdf.Ln(15)

	// Create a box for totals
	pdf.SetDrawColor(200, 200, 200)
	pdf.Rect(110, pdf.GetY(), 90, 40, "D")

	// Subtotal
	pdf.SetFont("Arial", "", 10)
	pdf.SetX(115)
	pdf.Cell(35, 12, "Subtotal:")
	currencySymbol := getCurrencySymbol(invoice.Currency)
	pdf.Cell(40, 12, fmt.Sprintf("%s%.2f", currencySymbol, subtotal))
	pdf.Ln(12)

	// Add subtle line
	pdf.SetDrawColor(220, 220, 220)
	pdf.Line(115, pdf.GetY(), 195, pdf.GetY())
	pdf.Ln(5)

	// Total
	pdf.SetFont("Arial", "B", 12)
	pdf.SetX(115)
	pdf.Cell(35, 12, "Total:")
	pdf.Cell(40, 12, fmt.Sprintf("%s%.2f", currencySymbol, subtotal))
	pdf.Ln(12)

	// Amount Due - make it stand out
	pdf.SetFillColor(245, 245, 245)
	pdf.Rect(110, pdf.GetY(), 90, 15, "F")
	pdf.SetFont("Arial", "B", 14)
	pdf.SetX(115)
	pdf.Cell(35, 15, "Amount Due:")
	pdf.SetTextColor(0, 100, 0) // Dark green color
	pdf.Cell(40, 15, fmt.Sprintf("%s%.2f", currencySymbol, subtotal))
	pdf.SetTextColor(0, 0, 0) // Reset to black
	pdf.Ln(20)

	// Generate PDF bytes
	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	return buf.Bytes(), nil
}

func (is *InvoiceService) SendInvoiceToSlack(userID, channelID string, invoice *models.InvoiceData, pdfBytes []byte) error {
	// Calculate total
	var total float64
	for _, item := range invoice.LineItems {
		total += float64(item.Quantity) * item.UnitPrice
	}

	// Create message
	message := fmt.Sprintf(
		"📄 *Invoice #%s* for *%s*\n\n*Amount Due:* $%.2f\n*Due Date:* %s\n*Email:* %s\n\nPlease find the PDF invoice attached.",
		invoice.InvoiceNumber, invoice.ClientName, total, invoice.DateDue, invoice.ClientEmail,
	)

	// Upload PDF to Slack
	uploadParams := slack.FileUploadParameters{
		Reader:         bytes.NewReader(pdfBytes),
		Filename:       fmt.Sprintf("Invoice_%s.pdf", invoice.InvoiceNumber),
		Title:          fmt.Sprintf("Invoice %s", invoice.InvoiceNumber),
		Filetype:       "pdf",
		Channels:       []string{channelID},
		InitialComment: message,
	}

	_, err := is.slackClient.UploadFile(uploadParams)
	if err != nil {
		log.Printf("Error uploading invoice to channel %s: %v", channelID, err)

		// Fallback: send to user's DM with debug note
		debugMessage := message + fmt.Sprintf("\n\n:warning: _This file was not sent to the channel because of: %v. Perhaps add the bot to the channel?_", err)
		dmUploadParams := slack.FileUploadParameters{
			Reader:         bytes.NewReader(pdfBytes),
			Filename:       fmt.Sprintf("Invoice_%s.pdf", invoice.InvoiceNumber),
			Title:          fmt.Sprintf("Invoice %s", invoice.InvoiceNumber),
			Filetype:       "pdf",
			InitialComment: debugMessage,
		}

		_, dmErr := is.slackClient.UploadFile(dmUploadParams)
		if dmErr != nil {
			return fmt.Errorf("failed to upload invoice to both channel and DM: %v (channel error: %v)", dmErr, err)
		}
	}

	return nil
}

func (is *InvoiceService) ParseInvoiceDataFromModal(values map[string]map[string]slack.BlockAction) (*models.InvoiceData, error) {
	invoice := &models.InvoiceData{
		LineItems: []models.InvoiceLineItem{},
	}

	// Parse basic fields
	invoice.InvoiceNumber = values["invoice_number_block"]["invoice_number_input"].Value
	invoice.ClientName = values["client_name_block"]["client_name_input"].Value
	invoice.ClientAddress = values["client_address_block"]["client_address_input"].Value
	invoice.ClientEmail = values["client_email_block"]["client_email_input"].Value
	invoice.DateDue = values["date_due_block"]["date_due_input"].Value

	// Parse currency (default to USD)
	if currencyBlock, exists := values["currency_block"]; exists {
		invoice.Currency = strings.ToUpper(strings.TrimSpace(currencyBlock["currency_input"].Value))
	}
	if invoice.Currency == "" {
		invoice.Currency = "USD"
	}

	// Parse line items from the new format
	lineItemsText := values["line_items_block"]["line_items_input"].Value
	if lineItemsText == "" {
		return nil, fmt.Errorf("at least one line item is required")
	}

	// Split by lines and parse each line item
	lines := strings.Split(strings.TrimSpace(lineItemsText), "\n")
	for lineNum, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue // Skip empty lines
		}

		// Parse line in format: "Service Description | Price | Quantity"
		parts := strings.Split(line, "|")
		if len(parts) < 2 {
			return nil, fmt.Errorf("line %d is not in the correct format. Expected: 'Service | Price | Quantity'", lineNum+1)
		}

		// Extract service description (everything before the first pipe)
		serviceDesc := strings.TrimSpace(parts[0])
		if serviceDesc == "" {
			return nil, fmt.Errorf("service description on line %d cannot be empty", lineNum+1)
		}

		// Extract price (second part)
		var unitPrice float64
		var err error
		if len(parts) >= 2 {
			priceStr := strings.TrimSpace(parts[1])
			unitPrice, err = strconv.ParseFloat(priceStr, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid price '%s' on line %d: %v", priceStr, lineNum+1, err)
			}
		}

		// Extract quantity (third part, optional - defaults to 1)
		quantity := 1
		if len(parts) >= 3 {
			quantityStr := strings.TrimSpace(parts[2])
			if quantityStr != "" {
				parsedQuantity, err := strconv.Atoi(quantityStr)
				if err != nil {
					return nil, fmt.Errorf("invalid quantity '%s' on line %d: %v", quantityStr, lineNum+1, err)
				}
				if parsedQuantity > 0 {
					quantity = parsedQuantity
				}
			}
		}

		invoice.LineItems = append(invoice.LineItems, models.InvoiceLineItem{
			ServiceDescription: serviceDesc,
			UnitPrice:          unitPrice,
			Quantity:           quantity,
		})
	}

	if len(invoice.LineItems) == 0 {
		return nil, fmt.Errorf("at least one valid line item is required")
	}

	return invoice, nil
}
