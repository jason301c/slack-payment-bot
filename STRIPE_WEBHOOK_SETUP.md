# Stripe Webhook Setup for EndDateCycles

This document explains how to set up Stripe webhooks to enable automatic subscription cancellation based on `EndDateCycles`.

## Overview

The implementation uses Stripe webhooks to:
1. Detect when a subscription is created from a payment link
2. Check if the subscription has cycle limits in metadata
3. Automatically schedule the subscription to cancel after the specified number of cycles

## Environment Variables

Add this new environment variable to your deployment:

```bash
STRIPE_WEBHOOK_SECRET=whsec_your_webhook_endpoint_secret_here
```

## Stripe Dashboard Setup

### 1. Create Webhook Endpoint

1. Go to [Stripe Dashboard](https://dashboard.stripe.com) → Developers → Webhooks
2. Click "Add endpoint"
3. Enter your endpoint URL: `https://yourdomain.com/stripe/webhook`
4. Select these events:
   - `customer.subscription.created`
   - `checkout.session.completed` (optional, for logging)

### 2. Get Webhook Secret

1. After creating the webhook, click on it
2. In the "Signing secret" section, click "Reveal"
3. Copy the secret (starts with `whsec_`)
4. Set it as the `STRIPE_WEBHOOK_SECRET` environment variable

### 3. Test Webhook

Test the webhook endpoint:
```bash
stripe listen --forward-to localhost:8080/stripe/webhook
```

## How It Works

### Payment Link Creation
When creating a subscription payment link with `EndDateCycles > 0`:

```go
// Example: 12 monthly payments
data := &models.PaymentLinkData{
    Amount:         29.99,
    ServiceName:    "Premium Service",
    IsSubscription: true,
    Interval:       "month",
    IntervalCount:  1,
    EndDateCycles:  12, // Will cancel after 12 months
}
```

### Metadata Storage
The system stores cycle information in Stripe subscription metadata:
- `end_date_cycles`: Number of cycles before cancellation
- `end_timestamp`: Unix timestamp when to cancel
- `interval`: Billing interval (month, week, year)
- `interval_count`: Interval multiplier

### Automatic Cancellation
When Stripe creates the subscription:
1. Webhook receives `customer.subscription.created` event
2. Handler checks for `end_date_cycles` in metadata
3. Schedules subscription to cancel using `cancel_at` parameter
4. Subscription automatically ends at the calculated time

## Testing

### Manual Test
1. Create a payment link with `EndDateCycles: 2`
2. Complete the checkout
3. Check Stripe Dashboard → Subscriptions
4. Verify the subscription shows "Cancels on [date]"

### Webhook Test
```bash
# Send test webhook event
stripe events resend evt_test_webhook --webhook-endpoint we_xxx
```

## Logging

The webhook handler logs important events:
- Subscription creation
- Cycle limit detection
- Cancellation scheduling
- Errors and failures

Check your application logs for entries like:
```
Subscription created: sub_1234567890
Scheduling subscription sub_1234567890 to cancel after 12 cycles
Successfully scheduled cancellation for subscription sub_1234567890
```

## Troubleshooting

### Common Issues

1. **Webhook not receiving events**
   - Verify endpoint URL is publicly accessible
   - Check webhook secret matches environment variable
   - Ensure events are selected in Stripe Dashboard

2. **Subscription not canceling**
   - Check application logs for errors
   - Verify metadata is being set correctly
   - Test timestamp calculation with different intervals

3. **Invalid webhook signature**
   - Verify `STRIPE_WEBHOOK_SECRET` environment variable
   - Check if webhook endpoint secret was copied correctly

### Debug Mode
Add debug logging to webhook handler:
```go
log.Printf("Received webhook event: %s", event.Type)
log.Printf("Subscription metadata: %+v", sub.Metadata)
```

## Security Notes

- Never expose webhook secrets in logs or source code
- Use HTTPS for production webhook endpoints
- Validate webhook signatures to prevent unauthorized requests
- Consider rate limiting webhook endpoints

## Limitations

- Uses approximate time calculations (30 days = 1 month, 365 days = 1 year)
- Requires webhook endpoint to be publicly accessible
- Cancellation is scheduled at subscription creation, not dynamically updated