# Slack Payment Link Bot

This Slack bot allows you to generate real Airwallex and Stripe payment links directly from your Slack workspace using slash commands and interactive modals.

## Client Setup

### Slack App Setup

1. **Create a Slack App**
   - Go to [Slack API Apps](https://api.slack.com/apps) and click "Create New App" > "From scratch".
   - Name your app (e.g., "Payment Link Bot") and select your workspace.
   - Click "Create App".

2. **Configure Slash Commands**
   - In your app settings, go to **Features > Slash Commands**.
   - Create two commands:
     - `/create-airwallex-link` (Request URL: `https://YOUR_PUBLIC_URL/slack/commands`)
     - `/create-stripe-link` (Request URL: `https://YOUR_PUBLIC_URL/slack/commands`)
   - `YOUR_PUBLIC_URL` should be the URL where your bot server is hosted.
   - **Note:** You no longer provide arguments directly in the slash command. The bot will always open a modal for you to fill in the payment details.

3. **Install the App to Your Workspace**
   - Go to **Settings > Install App**.
   - Click "Install to YOUR COMPANY" and grant permissions.
   - Copy your **Bot User OAuth Token** and **Signing Secret** (you'll need to provide these to the server/bot operator).
   - (Note that the signing secret is from **Settings > Basic Information**).

4. **Share Credentials**
   - Provide the following to the person running the bot:
     - Bot User OAuth Token
     - Signing Secret

### Credentials Setup (used for server)
 - Go to Stripe's [dashboard](https://dashboard.stripe.com) and copy the **Secret Key** from the API section. This key will be used to create payment links.
 - For Stripe, you need *Payment Links (write)*, *Products (write)*, *Plan (write)*, *Prices (write)* and *Features (write)*.

 - Go to Airwallex's dashboard and create a **Restricted API key** with permissions to create links. This will be used to generate payment links.

#### Total Credentials Needed
 - **Bot User OAuth Token** from Slack App
 - **Signing Secret** from Slack App
 - **Stripe Secret Key** from Stripe Dashboard
 - **Airwallex Client ID** from Airwallex
 - **Airwallex API Key** from Airwallex

## For the Server: Bot Deployment & Environment

1. **Clone the Repository**
   - Download or clone the project files to your server or local machine.

2. **Prepare Environment Variables**
   - Create a `.env` file in the project root with the following (replace values as needed):
     ```
     SLACK_BOT_TOKEN='xoxb-YOUR-BOT-TOKEN'
     SLACK_SIGNING_SECRET='YOUR-SIGNING-SECRET'
     STRIPE_API_KEY='sk_test_YOUR_STRIPE_SECRET_KEY'
     AIRWALLEX_CLIENT_ID='YOUR_AIRWALLEX_CLIENT_ID'
     AIRWALLEX_API_KEY='YOUR_AIRWALLEX_API_KEY'
     PORT='8080' # Optional, defaults to this
     AIRWALLEX_BASE_URL='https://api.airwallex.com' # Optional, defaults to this
     ```

3. **Install Go and Dependencies, then run**
   - Ensure Go 1.16 or higher is installed.
   - Run:
     ```
     go mod tidy
     go run main.go
     ```

## Running with Docker

You can also run the bot using Docker (recommended for deployment):

1. **Build the Docker image**
   ```
   docker build -t slack-payment-bot .
   ```

2. **Run the Docker container**
   - Pass your environment variables using a `.env` file or directly with `-e` flags.
   - Example using a `.env` file:
     ```
     docker run --env-file .env -p 8080:8080 slack-payment-bot
     ```

3. **Access the Bot**
   - The bot will be accessible at `http://YOUR_BASE_URL/slack/commands` or your specified port.
   - Point the Slack app to this URL.

## Usage
- In your Slack workspace, use the slash commands:
  - `/create-airwallex-link`
  - `/create-stripe-link`
- The bot will open a modal for you to fill in the payment details (amount, service name, reference, and for Stripe, subscription options).
- After submitting the modal, the bot will respond with a real payment link for the requested provider.

## Stripe Recurring/Subscription Payments
You can create recurring (subscription) payment links with Stripe by selecting the subscription options in the modal. The modal will allow you to choose the billing interval and frequency.

## Notes
- **The bot must be invited to any channel you want it to post in.** Use `/invite @your-bot-name` in the channel.
- Ensure your server is publicly accessible for Slack to send requests.
- This server should be available at YOUR_BASE_URL. This URL would be used in Slack App settings for the slash commands and interactivity.
- **Direct argument parsing in slash commands is no longer supported.** All input is via the modal.

## Required Slack Bot Scopes
- `chat:write` (to post messages)
- `chat:write.public` (optional, to post in channels the bot isn't a member of)
- `commands` (for slash commands)
- For private channels: `groups:write` (if you want to post in private channels)

## Troubleshooting

### Bot can't post messages (`missing_scope`)
- Make sure your bot has the `chat:write` scope (and `chat:write.public` if needed).
- After adding scopes, reinstall the app to your workspace.

### Bot can't post to a channel (`channel_not_found`)
- Make sure the bot is invited to the channel with `/invite @your-bot-name`.
- For private channels, the bot must be a member and may need `groups:write` scope.
- If you want the bot to always DM the user, you can update the code to use the user ID as the channel.

### Modal won't open (`invalid_arguments`)
- Slack modal titles must be 24 characters or fewer.
- All block IDs and action IDs must be unique and non-empty.
- All required fields must be present in the modal JSON.

### Slack can't reach your server (`We're having trouble connecting. Try again?`)
- Your server must be publicly accessible (use [ngrok](https://ngrok.com/) or similar for local development).
- The Interactivity Request URL in your Slack app must be correct and use HTTPS.
- Your handler must respond within 3 seconds.

### Airwallex authentication fails
- Airwallex may return 201 Created instead of 200 OK for authentication. The bot now accepts both as success.

---

For any other issues, check your server logs and Slack app logs for more details.