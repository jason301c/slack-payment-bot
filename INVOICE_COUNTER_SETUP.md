# Invoice Counter Setup

## Overview
The invoice bot now automatically increments invoice numbers to prevent duplicate invoices. It uses Slack as the state storage by looking for a dedicated channel to store the last used invoice number.

## Setup Instructions

### Option 1: Create a Dedicated Counter Channel (Recommended)

1. **Create a new Slack channel** named `invoice-counter` or `invoice_bot_counter`
2. **Make the bot a member** of this channel
3. **Post the starting invoice number** as a message in this channel (e.g., "1000")

The bot will automatically:
- Look for this channel when generating invoices
- Read the latest message to get the last invoice number
- Post the new invoice number after each successful invoice generation

### Option 2: No Setup Required

If no counter channel is found, the bot will:
- Default to starting invoice number 1000
- Increment this number in memory during the session
- Not persist the counter across restarts

## How It Works

1. **When opening the invoice modal**: The bot looks for the counter channel and reads the last invoice number
2. **In the modal form**:
   - The auto-generated invoice number is displayed prominently at the top
   - The override field is empty by default and labeled as "Advanced"
   - Users can leave the override field empty to use the auto-generated number
   - Users can manually enter a number in the override field if needed
3. **After successful invoice generation**: The bot posts the new invoice number to the counter channel
4. **Next invoice**: Will use the incremented number

## Benefits

- ✅ Prevents duplicate invoice numbers
- ✅ Automatic increment reduces manual errors
- ✅ Persistent state using Slack as storage
- ✅ Works across multiple users and bot restarts
- ✅ Still allows manual override if needed

## Troubleshooting

If the auto-increment isn't working:
1. Ensure the bot is in the `invoice-counter` channel
2. Check that the channel has at least one message with a number
3. Verify the bot has posting permissions in the channel
4. Check the bot logs for any error messages