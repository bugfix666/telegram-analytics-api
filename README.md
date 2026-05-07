# Telegram Analytics API

## Service Overview

This service provides a REST API for fetching analytics from Telegram channels. It connects to Telegram via MTProto (using the `gotd/td` library) **as a user account** (not a bot) because only user accounts have access to full message history, views, and reactions.

## Features

- Analytics for the last N messages (`limit` param)
- Optional channel description keyword check (`keyword` param)
- Average views and average reactions
- Engagement rate = (reactions + forwards) / views * 100%
- Audience activity = (avg views / subscribers) * 100%
- Send messages to a channel/group
- Single endpoint for all analytics: `/get/`

### Technical Details

- **Auth** – on first run the service asks for a phone number and confirmation code, then saves a session file (`telegram.session`). Subsequent runs use the session automatically without re‑entering the code.
- **Proxy** – supports SOCKS5 proxy (set in `.env`) to bypass network blocks.
- **Testing** – the entire business layer is covered with unit tests using mocks (no real Telegram connection required).

## Requirements

- Go 1.23+
- Telegram API ID and API Hash (get from my.telegram.org/apps)
- A Telegram user account (not a bot)

## Installation & Setup

### 1. Clone the repo
```bash
git clone ...
cd telegram-analytics-api
```

### 2. Configure environment
Copy the example config:
```bash
cp .env.example .env
```

Edit `.env` and fill in the mandatory fields:
```env
TELEGRAM_API_ID=1234567
TELEGRAM_API_HASH=your_32_char_hash
TELEGRAM_SESSION_FILE=telegram.session
HTTP_PORT=8765
# TELEGRAM_PROXY=socks5://localhost:1080   # optional, if Telegram is blocked
# TELEGRAM_BOT_TOKEN=                      # leave empty for user account
```

Important: `TELEGRAM_BOT_TOKEN` must be empty; otherwise the service will log in as a bot and won’t be able to read message history.

### 3. Build & Run
Build the binary:
```bash
go build -o telegram-analytics cmd/server/main.go
```

Run the server:
```bash
./telegram-analytics
```

First run:
- The program will ask for your phone number (international format, e.g. `+79123456789`)
- Then ask for the confirmation code sent to your Telegram
- If 2FA is enabled, it will ask for your password
- After successful login, a session file (`telegram.session`) is created and reused on next runs

## API Endpoints

| Method | URL | Description |
|--------|-----|-------------|
| GET | `/get/?group_id=...&limit=N&keyword=...` | Full analytics for last N messages + optional keyword check |
| POST | `/send_message/` | Send a text message to a chat/channel |

### GET Parameters

- `group_id` (required) – channel username, with or without `@` (e.g., `@durov` or `durov`)
- `limit` (optional) – number of most recent messages to analyze (default: 1000)
- `keyword` (optional) – word to search in the channel description. If omitted, the `contains_keyword` field is not returned.

### Response Fields (GET /get/)

- `average_views` – average views per message
- `average_reactions` – average reactions (likes) per message
- `engagement_rate_percent` – (reactions + forwards) / views * 100%
- `messages_processed` – number of messages processed (min(limit, available messages))
- `subscribers` – channel subscriber count
- `activity_percentage` – (average_views / subscribers) * 100%
- `contains_keyword` – present only if `keyword` was passed; `true` if the description contains that keyword.

## Examples

### Get analytics for last 500 messages
```bash
curl "http://localhost:8765/get/?group_id=@durov&limit=500"
```

Response:
```json
{
  "average_views": 15230.5,
  "average_reactions": 45.2,
  "engagement_rate_percent": 0.36,
  "messages_processed": 500,
  "subscribers": 2500000,
  "activity_percentage": 0.61
}
```

### Check for a keyword in the description
```bash
curl "http://localhost:8765/get/?group_id=@durov&keyword=telegram"
```

Response (keyword not found):
```json
{
  "average_views": 14200.3,
  "average_reactions": 38.7,
  "engagement_rate_percent": 0.29,
  "messages_processed": 1000,
  "subscribers": 2500000,
  "activity_percentage": 0.57,
  "contains_keyword": false
}
```

### Send a message
```bash
curl -X POST http://localhost:8765/send_message/ \
  -H "Content-Type: application/json" \
  -d '{"chat_id":"@durov","text":"Hi from API!"}'
```

Response:
```json
{
  "message_id": 12345,
  "status": "sent"
}
```

Error response (e.g., chat not found):
```json
{
  "detail": "chat not found"
}
```

## Testing

Run all tests (uses mocks, no real Telegram connection needed):
```bash
go test ./... -v -cover
```

## Troubleshooting

| Problem | Solution |
|---------|----------|
| `CONNECTION_API_ID_INVALID` | Wrong API_ID or API_HASH. Get new ones from my.telegram.org/apps. |
| `context deadline exceeded` | Telegram is blocked. Use a proxy: set `TELEGRAM_PROXY=socks5://...` in `.env`. |
| `BOT_METHOD_INVALID` | You are using a bot. Remove `TELEGRAM_BOT_TOKEN` from `.env` and re-authenticate as a user. |
| `average_views` and reactions are zero | Most likely you're still using a bot. Delete `telegram.session` and `TELEGRAM_BOT_TOKEN`, then restart the server. |
| `.env` not loaded | Put `.env` in the same directory as the binary or from where you run `go run`. |