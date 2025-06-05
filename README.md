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

3. **Configure Interactivity & Shortcuts**
   - In your app settings, go to **Features > Interactivity & Shortcuts**.
   - Enable interactivity and set the Request URL to `https://YOUR_PUBLIC_URL/slack/interactions`.

4. **Install the App to Your Workspace**
   - Go to **Settings > Install App**.
   - Click "Install to YOUR COMPANY" and grant permissions.
   - Copy your **Bot User OAuth Token** and **Signing Secret** (you'll need to provide these to the server/bot operator).
   - (Note that the signing secret is from **Settings > Basic Information**).

5. **Share Credentials**
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
- Ensure your server is publicly accessible for Slack to send requests.
- This server should be available at YOUR_BASE_URL. This URL would be used in Slack App settings for the slash commands and interactivity.
- **Direct argument parsing in slash commands is no longer supported.** All input is via the modal.

---

For any other issues, check your server logs and Slack app logs for more details.