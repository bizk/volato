# API Registration Guide

Volato supports two flight search APIs. You need at least one configured to search for flights.

## Kiwi (Tequila) API

**Free tier:** 3,000 requests/month

### Registration Steps

1. Go to [Tequila by Kiwi.com](https://tequila.kiwi.com/)

2. Click "Sign up" and create an account

3. Verify your email address

4. Go to your dashboard and find your API key under "My solutions"

5. Copy the API key to your config:
   ```toml
   [apis.kiwi]
   api_key = "YOUR_API_KEY_HERE"
   ```

### Features

- Broad coverage of low-cost carriers
- Includes booking links in results
- Good for budget airlines and regional routes
- Date range search support

### API Documentation

- [Tequila API Docs](https://tequila.kiwi.com/portal/docs/tequila_api)
- [Search endpoint](https://tequila.kiwi.com/portal/docs/tequila_api/search_api)

---

## Amadeus API

**Free tier:** 2,000 calls/month

### Registration Steps

1. Go to [Amadeus for Developers](https://developers.amadeus.com/)

2. Click "Register" and create an account

3. Verify your email address

4. Create a new app in your dashboard:
   - Go to "My Self-Service Workspace"
   - Click "Create New App"
   - Name your app (e.g., "Volato")
   - Select "Test" environment for free tier

5. Copy the Client ID and Client Secret to your config:
   ```toml
   [apis.amadeus]
   client_id = "YOUR_CLIENT_ID"
   client_secret = "YOUR_CLIENT_SECRET"
   ```

### Features

- Access to major airline inventories
- Accurate pricing from airline sources
- Better coverage of full-service carriers
- More detailed flight information

### API Documentation

- [Amadeus API Docs](https://developers.amadeus.com/self-service/apis-docs)
- [Flight Offers Search](https://developers.amadeus.com/self-service/category/flights/api-doc/flight-offers-search)

### Note on Test vs Production

The free "Test" environment uses cached/sample data. For real-time pricing, you would need to apply for production access, which requires a business use case approval.

---

## Telegram Bot Setup

### Create a Bot

1. Open Telegram and search for [@BotFather](https://t.me/BotFather)

2. Start a conversation and send `/newbot`

3. Follow the prompts:
   - Enter a name for your bot (e.g., "Volato Flight Alerts")
   - Enter a username (must end in "bot", e.g., "volato_alerts_bot")

4. BotFather will give you a token like:
   ```
   123456789:ABCdefGHIjklMNOpqrsTUVwxyz
   ```

5. Copy the token to your config:
   ```toml
   [telegram]
   bot_token = "123456789:ABCdefGHIjklMNOpqrsTUVwxyz"
   ```

### Get Your Chat ID

1. Search for [@userinfobot](https://t.me/userinfobot) on Telegram

2. Start a conversation and send any message

3. The bot will reply with your user info, including your ID:
   ```
   Id: 123456789
   ```

4. Copy the ID to your config:
   ```toml
   [telegram]
   chat_id = "123456789"
   ```

### Start the Conversation

**Important:** You must start a conversation with your bot before it can send you messages.

1. Search for your bot by its username
2. Click "Start" or send `/start`
3. Now the bot can send you notifications

### Bot Commands

Once running, your bot supports these commands:

| Command   | Description                              |
|-----------|------------------------------------------|
| `/status` | Show last check time and deals found     |
| `/check`  | Trigger a manual flight check (1hr cooldown) |
| `/deals`  | Show recent deals found                  |
| `/help`   | Show available commands                  |

---

## Which API Should I Use?

| Factor | Kiwi | Amadeus |
|--------|------|---------|
| Free requests/month | 3,000 | 2,000 |
| Low-cost carriers | Better | Limited |
| Major airlines | Good | Better |
| Booking links | Yes | No |
| Real-time data | Yes | Test only* |

*Production access requires approval.

**Recommendation:** Start with Kiwi for the free tier's higher limits and booking links. Add Amadeus if you want broader coverage of major carriers.

You can configure both APIs and Volato will query both, giving you the best of both worlds.
