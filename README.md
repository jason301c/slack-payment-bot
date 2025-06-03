# Slack Payment Link Bot

This Slack bot allows you to generate real Airwallex and Stripe payment links directly from your Slack workspace using slash commands.

---

## For the Client: Slack App Setup

1. **Create a Slack App**
   - Go to [Slack API Apps](https://api.slack.com/apps) and click "Create New App" > "From scratch".
   - Name your app (e.g., "Payment Link Bot") and select your workspace.
   - Click "Create App".

2. **Configure Slash Commands**
   - In your app settings, go to **Features > Slash Commands**.
   - Create two commands:
     - `/create-airwallex-link` (Request URL: `https://YOUR_PUBLIC_URL/slack/commands`)
     - `/create-stripe-link` (Request URL: `https://YOUR_PUBLIC_URL/slack/commands`)
   - Usage hint: `[amount] [service_name] [reference_number]`

3. **Install the App to Your Workspace**
   - Go to **Settings > Basic Information**.
   - Click "Install to Workspace" and grant permissions.
   - Copy your **Bot User OAuth Token** and **Signing Secret** (you'll need to provide these to the server/bot operator).

4. **Share Credentials**
   - Provide the following to the person running the bot:
     - Bot User OAuth Token
     - Signing Secret

---

## For the Server: Bot Deployment & Environment

1. **Clone the Repository**
   - Download or clone the project files to your server or local machine.

2. **Prepare Environment Variables**
   - Create a `.env` file in the project root with the following (replace values as needed):
     ```
     SLACK_BOT_TOKEN='xoxb-YOUR-BOT-TOKEN'
     SLACK_SIGNING_SECRET='YOUR-SIGNING-SECRET'
     PORT='8080'
     STRIPE_API_KEY='sk_test_YOUR_STRIPE_SECRET_KEY'
     AIRWALLEX_CLIENT_ID='YOUR_AIRWALLEX_CLIENT_ID'
     AIRWALLEX_API_KEY='YOUR_AIRWALLEX_API_KEY'
     AIRWALLEX_BASE_URL='https://api-demo.airwallex.com' # or your production endpoint
     ```

3. **Install Go and Dependencies**
   - Ensure Go 1.16 or higher is installed.
   - Run:
     ```
     go mod tidy
     ```

4. **(Optional) Expose Locally with Ngrok**
   - If running locally, use [Ngrok](https://ngrok.com/download) to expose your port:
     ```
     ./ngrok http 8080
     ```
   - Update the Slack command URLs to your Ngrok HTTPS URL (e.g., `https://abcdef12345.ngrok.io/slack/commands`).

5. **Run the Bot**
   - Start the bot:
     ```
     go run main.go
     ```

---

## Usage
- In your Slack workspace, use the slash commands:
  - `/create-airwallex-link 100.50 "Website Design" INV-2023-001`
  - `/create-stripe-link 250.00 "Consulting Service" REF-ABC-XYZ`
- The bot will respond with a real payment link for the requested provider.