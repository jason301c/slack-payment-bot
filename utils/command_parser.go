package utils

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"paymentbot/models"
)

// SplitArgsQuoted splits a command string into arguments, treating quoted substrings as single arguments.
func SplitArgsQuoted(input string) []string {
	var args []string
	var current strings.Builder
	inQuotes := false
	var quoteChar rune

	for _, r := range input {
		switch {
		case r == '"' || r == '\'':
			if !inQuotes {
				inQuotes = true
				quoteChar = r
			} else if r == quoteChar {
				inQuotes = false
				if current.Len() > 0 {
					args = append(args, current.String())
					current.Reset()
				}
			} else {
				current.WriteRune(r)
			}
		case r == ' ' && !inQuotes:
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args
}

// ParseCommandArguments parses the text from a Slack slash command.
// Format: <amount> "<service_name>" <reference_number>
func ParseCommandArguments(text string) (*models.PaymentLinkData, error) {
	parts := SplitArgsQuoted(text)

	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid format. Usage: <amount> \"<service_name>\" [reference_number]")
	}

	// Parse amount
	amountStr := strings.TrimSpace(parts[0])
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid amount '%s'. Please provide a valid number", amountStr)
	}
	if amount <= 0 {
		return nil, fmt.Errorf("amount must be greater than 0")
	}

	// Get service name
	serviceName := strings.TrimSpace(parts[1])
	if serviceName == "" {
		return nil, fmt.Errorf("service name cannot be empty")
	}

	// Get reference number (optional)
	referenceNumber := fmt.Sprintf("REF-%d", time.Now().Unix())
	if len(parts) > 2 {
		referenceNumber = strings.TrimSpace(parts[2])
	}

	// Parse subscription options if provided
	isSubscription := false
	interval := "month"
	intervalCount := int64(1)

	if len(parts) > 3 {
		subStr := strings.ToLower(strings.TrimSpace(parts[3]))
		isSubscription = subStr == "true" || subStr == "yes" || subStr == "1"

		if isSubscription {
			if len(parts) > 4 {
				interval = strings.ToLower(strings.TrimSpace(parts[4]))
				if !IsValidInterval(interval) {
					return nil, fmt.Errorf("invalid interval '%s'. Must be one of: month, week, year", interval)
				}
			}

			if len(parts) > 5 {
				count, err := strconv.ParseInt(strings.TrimSpace(parts[5]), 10, 64)
				if err != nil {
					return nil, fmt.Errorf("invalid interval count '%s'. Must be a positive number", parts[5])
				}
				if count < 1 {
					return nil, fmt.Errorf("interval count must be greater than 0")
				}
				intervalCount = count
			}
		}
	}

	return &models.PaymentLinkData{
		Amount:          amount,
		ServiceName:     serviceName,
		ReferenceNumber: referenceNumber,
		IsSubscription:  isSubscription,
		Interval:        interval,
		IntervalCount:   intervalCount,
	}, nil
}

// IsValidInterval checks if the provided interval is valid
func IsValidInterval(interval string) bool {
	validIntervals := map[string]bool{
		"month": true,
		"week":  true,
		"year":  true,
	}
	return validIntervals[interval]
}
