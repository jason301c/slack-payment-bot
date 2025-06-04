package main

import (
	"fmt"
	"strconv"
	"strings"
)

// splitArgsQuoted splits a command string into arguments, treating quoted substrings or bracketed substrings as single arguments.
func splitArgsQuoted(input string) []string {
	var args []string
	var current strings.Builder
	inGroup := false
	var groupChar rune

	for _, r := range input {
		if inGroup {
			if r == groupChar || (groupChar == '[' && r == ']') {
				inGroup = false
				args = append(args, current.String())
				current.Reset()
				continue
			}
			current.WriteRune(r)
			continue
		}
		if r == '"' || r == '\'' || r == '[' {
			inGroup = true
			if r == '[' {
				groupChar = '['
			} else {
				groupChar = r
			}
			continue
		}
		if r == ' ' || r == '\t' {
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
			continue
		}
		current.WriteRune(r)
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	// If inGroup is still true, treat as unterminated group, add as is
	if inGroup && current.Len() > 0 {
		args = append(args, current.String())
	}
	return args
}

// parseCommandArguments parses the text from a Slack slash command.
// It expects the format: "[amount] [service_name] [reference_number]"
// It also supports quoted strings and bracketed strings.
func parseCommandArguments(text string) (*PaymentLinkData, error) {
	parts := splitArgsQuoted(text)

	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid number of arguments. Usage: [amount] [service_name] [reference_number]")
	}

	amountStr := parts[0]
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid amount. Ensure the amount is a positive number (in USD)")
	}
	if amount <= 0 {
		return nil, fmt.Errorf("amount must be a positive number")
	}

	serviceName := parts[1]
	referenceNumber := parts[2]

	return &PaymentLinkData{
		Amount:          amount,
		ServiceName:     serviceName,
		ReferenceNumber: referenceNumber,
	}, nil
}
