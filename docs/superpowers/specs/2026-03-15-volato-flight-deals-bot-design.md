# Volato Flight Deals Bot - Design Specification

**Date:** 2026-03-15
**Status:** Approved

## Overview

Volato is a Go CLI tool that monitors flight prices from Argentina and sends Telegram notifications when deals are found. It queries multiple flight APIs, detects deals based on price thresholds and historical price drops, and provides basic Telegram bot commands for interaction.

## Requirements

### Functional Requirements

1. **Flight Search**
   - Query flights from configurable origin airports (default: Buenos Aires EZE/AEP)
   - Search multiple destinations with configurable parameters
   - Support flexible date ranges and stay durations

2. **Deal Detection**
   - Alert when price falls below configured maximum
   - Alert when price drops 20%+ from 30-day average (configurable threshold)
   - Avoid duplicate alerts for the same deal within 24 hours

3. **Notifications**
   - Send Telegram messages with: route, dates, price, booking link
   - Support bot commands: `/status`, `/check`, `/deals`

4. **Scheduling**
   - Run as daily cron job for automatic checks
   - Support manual triggering via Telegram `/check` command

### Non-Functional Requirements

- Deploy on Raspberry Pi 5
- Use free tier APIs (Kiwi: 3000/month, Amadeus: 2000/month)
- SQLite for storage (no external database dependencies)
- Single binary deployment

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         volato                              │
├─────────────────────────────────────────────────────────────┤
│  Commands:                                                  │
│    volato check   ─► Fetch flights, detect deals, notify   │
│    volato bot     ─► Listen for Telegram commands          │
│    volato migrate ─► Initialize SQLite database            │
└─────────────────────────────────────────────────────────────┘
            │                           │
            ▼                           ▼
┌───────────────────┐         ┌───────────────────┐
│   Flight APIs     │         │   Telegram API    │
│  ┌─────────────┐  │         │                   │
│  │ Kiwi/Tequila│  │         │  • Send alerts    │
│  └─────────────┘  │         │  • /status        │
│  ┌─────────────┐  │         │  • /check         │
│  │   Amadeus   │  │         │  • /deals         │
│  └─────────────┘  │         │                   │
└───────────────────┘         └───────────────────┘
            │                           │
            └───────────┬───────────────┘
                        ▼
              ┌───────────────────┐
              │  SQLite Database  │
              │  • price_history  │
              │  • sent_deals     │
              └───────────────────┘
```

### Data Flow

1. Cron runs `volato check` daily
2. Check loads TOML config
3. For each search: query Kiwi, then Amadeus
4. Merge and deduplicate results (keep lowest price per route+dates)
5. Compare against thresholds and price history
6. Send Telegram alerts for deals found
7. Store prices in SQLite for future drop detection

## Project Structure

```
volato/
├── cmd/
│   └── volato/
│       └── main.go           # CLI entry point (cobra commands)
├── internal/
│   ├── config/
│   │   └── config.go         # TOML parsing, validation
│   ├── api/
│   │   ├── client.go         # Common interface for flight APIs
│   │   ├── kiwi.go           # Kiwi/Tequila implementation
│   │   └── amadeus.go        # Amadeus implementation
│   ├── deals/
│   │   └── detector.go       # Price threshold + drop detection logic
│   ├── storage/
│   │   └── sqlite.go         # SQLite operations
│   ├── telegram/
│   │   ├── bot.go            # Bot listener for commands
│   │   └── notifier.go       # Send deal alerts
│   └── checker/
│       └── checker.go        # Orchestrates check flow
├── migrations/
│   └── 001_initial.sql       # Database schema
├── docs/
│   ├── setup.md              # Raspberry Pi installation guide
│   ├── apis.md               # API registration steps
│   └── configuration.md      # Config file reference
├── config.example.toml       # Example configuration
├── go.mod
└── go.sum
```

## Configuration

Location: `~/.config/volato/config.toml`

```toml
# Telegram settings
[telegram]
bot_token = "123456:ABC-DEF..."
chat_id = "your_chat_id"

# API credentials
[apis.kiwi]
api_key = "your_kiwi_api_key"

[apis.amadeus]
client_id = "your_amadeus_client_id"
client_secret = "your_amadeus_client_secret"

# Default origin (can be overridden per search)
[defaults]
origin = "EZE"  # Buenos Aires Ezeiza
currency = "USD"

# Price drop detection
[alerts]
drop_threshold_percent = 20  # Alert if price drops 20% from average

# Searches - define each destination you want to track
[[searches]]
destination = "BCN"  # Barcelona
months = [6, 7, 8]   # June, July, August
stay_days = { min = 10, max = 14 }
max_price = 900

[[searches]]
destination = "FCO"  # Rome
origin = "AEP"       # Override: use Aeroparque
months = [3, 4, 5]
stay_days = { min = 7, max = 10 }
max_price = 850
```

### Search Configuration Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `destination` | string | Yes | IATA airport code |
| `origin` | string | No | Override default origin |
| `months` | int[] | Yes | Month numbers to search (1-12) |
| `stay_days` | object | Yes | `{min, max}` trip duration range |
| `max_price` | float | Yes | Maximum price threshold |

### Future Filters (Extensible)

```toml
[searches.filters]
max_stops = 1
airlines = ["AA", "LA"]
exclude_airlines = ["FR"]
cabin_class = "economy"
direct_only = false
```

## Database Schema

Location: `~/.local/share/volato/volato.db`

```sql
-- Track price history for drop detection
CREATE TABLE price_history (
    id INTEGER PRIMARY KEY,
    origin TEXT NOT NULL,
    destination TEXT NOT NULL,
    departure_date TEXT NOT NULL,      -- YYYY-MM-DD
    return_date TEXT NOT NULL,         -- YYYY-MM-DD
    price REAL NOT NULL,
    currency TEXT NOT NULL,
    api_source TEXT NOT NULL,          -- 'kiwi' or 'amadeus'
    checked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Avoid sending duplicate alerts for the same deal
CREATE TABLE sent_deals (
    id INTEGER PRIMARY KEY,
    flight_hash TEXT UNIQUE NOT NULL,  -- Hash of route+dates+price
    sent_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Index for efficient price history queries
CREATE INDEX idx_price_history_route
ON price_history(origin, destination, departure_date, return_date);
```

### Deal Deduplication

`flight_hash` is computed as: `SHA256(origin + destination + departure_date + return_date + price_bucket)`

Price bucket rounds to nearest $10 to avoid alerting on minor fluctuations.

## API Integration

### Common Interface

```go
type FlightSearcher interface {
    Search(ctx context.Context, req SearchRequest) ([]Flight, error)
    Name() string
}

type SearchRequest struct {
    Origin      string
    Destination string
    DateFrom    time.Time
    DateTo      time.Time
    StayDaysMin int
    StayDaysMax int
    Currency    string
    Filters     *SearchFilters
}

type SearchFilters struct {
    MaxStops        *int
    Airlines        []string
    ExcludeAirlines []string
    CabinClass      *string
    DirectOnly      bool
}

type Flight struct {
    Origin        string
    Destination   string
    DepartureDate time.Time
    ReturnDate    time.Time
    Price         float64
    Currency      string
    Airline       string
    Stops         int
    BookingLink   string
    APISource     string
}
```

### Kiwi (Tequila) API

- **Endpoint:** `https://api.tequila.kiwi.com/v2/search`
- **Auth:** API key in header (`apikey`)
- **Free tier:** 3000 requests/month
- **Features:** Native date range support, booking deep links

### Amadeus API

- **Endpoint:** `https://api.amadeus.com/v2/shopping/flight-offers`
- **Auth:** OAuth 2.0 (client credentials flow)
- **Free tier:** 2000 calls/month
- **Features:** Broader airline coverage

### Query Strategy

1. Query Kiwi first (primary source)
2. Query Amadeus for same parameters
3. Merge results
4. Deduplicate by route + dates, keeping lowest price
5. Return unified `[]Flight` slice

## Telegram Integration

### Notification Format

```
✈️ Deal Found!

🛫 Buenos Aires (EZE) → Barcelona (BCN)
📅 Jun 15 - Jun 25, 2025 (10 days)
💰 $745 USD

🔗 Book: https://kiwi.com/...
```

### Bot Commands

| Command | Description |
|---------|-------------|
| `/status` | Bot status, last check time, API quota usage |
| `/check` | Trigger immediate flight check |
| `/deals` | List last 10 deals found |

### Dependencies

- `github.com/go-telegram-bot-api/telegram-bot-api/v5`

## Deployment

### Target Environment

- Raspberry Pi 5 with Raspberry Pi OS (64-bit)
- Go 1.21+
- SQLite 3

### Components

1. **Binary:** `/usr/local/bin/volato`
2. **Config:** `~/.config/volato/config.toml`
3. **Database:** `~/.local/share/volato/volato.db`
4. **Systemd service:** `/etc/systemd/system/volato-bot.service` (for Telegram bot)
5. **Cron job:** Daily at 8 AM for flight checks

### Systemd Service

```ini
[Unit]
Description=Volato Telegram Bot
After=network.target

[Service]
Type=simple
User=pi
ExecStart=/usr/local/bin/volato bot
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

### Cron Entry

```
0 8 * * * /usr/local/bin/volato check >> /var/log/volato.log 2>&1
```

## Documentation Deliverables

| File | Contents |
|------|----------|
| `docs/setup.md` | Raspberry Pi installation, systemd, cron setup |
| `docs/apis.md` | API registration steps for Kiwi, Amadeus, Telegram |
| `docs/configuration.md` | Config file reference with all options |

## Error Handling

- API failures: Log error, continue with other API
- Both APIs fail: Log error, send Telegram alert about check failure
- Telegram send failure: Retry 3 times with exponential backoff
- Database errors: Log and exit with non-zero code
- Rate limiting: `/check` command has a 1-hour cooldown to prevent API quota exhaustion

## Data Retention

- Price history: Delete records older than 90 days (cleanup runs during each check)
- Sent deals: Delete records older than 7 days
- API quota tracking: Simple in-memory counter reset on first check of each month

## Testing Strategy

- Unit tests for deal detection logic
- Unit tests for config parsing
- Integration tests with mock API responses
- Manual end-to-end testing with real APIs

## Future Considerations

- Additional flight APIs (Google Flights, etc.)
- Full Telegram management commands (add/remove searches)
- Web dashboard for viewing deal history
- Support for one-way flights
- Multi-city trip searches
