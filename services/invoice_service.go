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

func (is *InvoiceService) GenerateInvoicePDF(invoice *models.InvoiceData) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	// Set font
	pdf.SetFont("Arial", "", 12)

	// Company header
	pdf.SetFont("Arial", "B", 20)
	pdf.Cell(0, 10, "INVOICE")
	pdf.Ln(15)

	// Invoice number and date
	pdf.SetFont("Arial", "", 12)
	pdf.Cell(95, 8, fmt.Sprintf("Invoice Number: %s", invoice.InvoiceNumber))
	pdf.Cell(95, 8, fmt.Sprintf("Date: %s", time.Now().Format("January 2, 2006")))
	pdf.Ln(8)
	pdf.Cell(95, 8, fmt.Sprintf("Due Date: %s", invoice.DateDue))
	pdf.Ln(20)

	// Bill To section
	pdf.SetFont("Arial", "B", 14)
	pdf.Cell(0, 8, "Bill To:")
	pdf.Ln(8)

	pdf.SetFont("Arial", "", 12)
	pdf.Cell(0, 6, invoice.ClientName)
	pdf.Ln(6)
	if invoice.ClientAddress != "" {
		pdf.Cell(0, 6, invoice.ClientAddress)
		pdf.Ln(6)
	}
	if invoice.ClientEmail != "" {
		pdf.Cell(0, 6, invoice.ClientEmail)
		pdf.Ln(15)
	} else {
		pdf.Ln(9)
	}

	// Table headers
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(100, 10, "Description")
	pdf.Cell(30, 10, "Quantity")
	pdf.Cell(60, 10, "Price")
	pdf.Ln(10)

	// Table line
	pdf.Line(10, pdf.GetY(), 200, pdf.GetY())
	pdf.Ln(5)

	// Line items
	pdf.SetFont("Arial", "", 12)
	var total float64
	for i, item := range invoice.LineItems {
		// Description
		pdf.Cell(100, 8, item.ServiceDescription)

		// Quantity
		quantity := fmt.Sprintf("%d", item.Quantity)
		pdf.Cell(30, 8, quantity)

		// Price
		lineTotal := float64(item.Quantity) * item.UnitPrice
		price := fmt.Sprintf("$%.2f", lineTotal)
		pdf.Cell(60, 8, price)
		pdf.Ln(8)

		total += lineTotal

		// Add spacing between items
		if i < len(invoice.LineItems)-1 {
			pdf.Ln(2)
		}
	}

	// Total line
	pdf.Ln(10)
	pdf.Line(10, pdf.GetY(), 200, pdf.GetY())
	pdf.Ln(5)

	pdf.SetFont("Arial", "B", 14)
	pdf.Cell(130, 10, "Total:")
	pdf.Cell(60, 10, fmt.Sprintf("$%.2f", total))
	pdf.Ln(20)

	// Footer notes
	pdf.SetFont("Arial", "I", 10)
	pdf.Cell(0, 8, "Thank you for your business!")
	pdf.Ln(6)
	pdf.Cell(0, 8, "Payment is due within 30 days.")

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
		Reader:          bytes.NewReader(pdfBytes),
		Filename:        fmt.Sprintf("Invoice_%s.pdf", invoice.InvoiceNumber),
		Title:           fmt.Sprintf("Invoice %s", invoice.InvoiceNumber),
		Filetype:        "pdf",
		Channels:        []string{channelID},
		InitialComment:  message,
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
			UnitPrice:         unitPrice,
			Quantity:          quantity,
		})
	}

	if len(invoice.LineItems) == 0 {
		return nil, fmt.Errorf("at least one valid line item is required")
	}

	return invoice, nil
}