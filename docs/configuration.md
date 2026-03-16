# Configuration Reference

Volato uses a TOML configuration file. By default, it looks for `~/.config/volato/config.toml`.

## Configuration Sections

### `[telegram]` - Telegram Bot Settings

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `bot_token` | string | Yes | Bot token from @BotFather |
| `chat_id` | string | Yes | Your Telegram user ID (numeric) |

```toml
[telegram]
bot_token = "123456789:ABCdefGHIjklMNOpqrsTUVwxyz"
chat_id = "123456789"
```

### `[apis.kiwi]` - Kiwi/Tequila API

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `api_key` | string | No* | API key from Tequila dashboard |

*At least one API (Kiwi or Amadeus) must be configured.

```toml
[apis.kiwi]
api_key = "your_kiwi_api_key"
```

### `[apis.amadeus]` - Amadeus API

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `client_id` | string | No* | Client ID from Amadeus dashboard |
| `client_secret` | string | No* | Client secret from Amadeus dashboard |

*At least one API (Kiwi or Amadeus) must be configured.

```toml
[apis.amadeus]
client_id = "your_client_id"
client_secret = "your_client_secret"
```

### `[defaults]` - Default Settings

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `origin` | string | Yes | - | Default origin airport (IATA code) |
| `currency` | string | Yes | - | Currency for prices (ISO 4217) |

```toml
[defaults]
origin = "EZE"
currency = "USD"
```

### `[alerts]` - Alert Settings

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `drop_threshold_percent` | float | No | 0 | Alert when price drops this % below average |

```toml
[alerts]
drop_threshold_percent = 20
```

### `[[searches]]` - Flight Searches

Each `[[searches]]` block defines a destination to monitor. You can have multiple searches.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `destination` | string | Yes | Destination airport (IATA code) |
| `origin` | string | No | Override default origin |
| `months` | array of strings | Yes | Months to search (1-12) |
| `max_price` | float | Yes | Maximum price threshold |
| `stay_days.min` | int | Yes | Minimum stay duration |
| `stay_days.max` | int | Yes | Maximum stay duration |

```toml
[[searches]]
destination = "BCN"
months = ["6", "7", "8"]
max_price = 900

[searches.stay_days]
min = 10
max = 14

[[searches]]
destination = "FCO"
origin = "AEP"  # Override: use different origin
months = ["3", "4", "5"]
max_price = 850

[searches.stay_days]
min = 7
max = 10
```

## CLI Flags

### Global Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--config` | Path to config file | `~/.config/volato/config.toml` |
| `--db` | Path to database file | `~/.local/share/volato/volato.db` |

### Commands

#### `volato check`

Run flight check and send deal notifications.

```bash
volato check
volato check --config /path/to/config.toml
volato check --db /path/to/volato.db
```

#### `volato bot`

Start Telegram bot listener.

```bash
volato bot
volato bot --config /path/to/config.toml
```

#### `volato migrate`

Initialize or migrate the database.

```bash
volato migrate
volato migrate --db /path/to/volato.db
```

## Common Airport Codes

### Argentina

| Code | Airport |
|------|---------|
| EZE | Buenos Aires Ezeiza International |
| AEP | Buenos Aires Aeroparque |
| COR | Cordoba |
| MDZ | Mendoza |
| BRC | Bariloche |
| IGR | Iguazu |

### Europe

| Code | Airport |
|------|---------|
| BCN | Barcelona |
| MAD | Madrid |
| FCO | Rome Fiumicino |
| CDG | Paris Charles de Gaulle |
| LHR | London Heathrow |
| FRA | Frankfurt |
| AMS | Amsterdam |
| LIS | Lisbon |

### Americas

| Code | Airport |
|------|---------|
| MIA | Miami |
| JFK | New York JFK |
| LAX | Los Angeles |
| GRU | Sao Paulo |
| SCL | Santiago Chile |
| BOG | Bogota |
| LIM | Lima |
| MEX | Mexico City |

### Other

| Code | Airport |
|------|---------|
| DXB | Dubai |
| TLV | Tel Aviv |
| NRT | Tokyo Narita |
| SYD | Sydney |

## Example Complete Configuration

```toml
# Telegram settings
[telegram]
bot_token = "123456789:ABCdefGHIjklMNOpqrsTUVwxyz"
chat_id = "987654321"

# API credentials
[apis.kiwi]
api_key = "your_kiwi_api_key"

[apis.amadeus]
client_id = "your_amadeus_client_id"
client_secret = "your_amadeus_client_secret"

# Defaults
[defaults]
origin = "EZE"
currency = "USD"

# Alert thresholds
[alerts]
drop_threshold_percent = 20

# Summer Europe trip
[[searches]]
destination = "BCN"
months = ["6", "7", "8"]
max_price = 900

[searches.stay_days]
min = 10
max = 14

# Spring Italy
[[searches]]
destination = "FCO"
months = ["3", "4", "5"]
max_price = 850

[searches.stay_days]
min = 7
max = 10

# Winter Miami from Aeroparque
[[searches]]
destination = "MIA"
origin = "AEP"
months = ["12", "1", "2"]
max_price = 600

[searches.stay_days]
min = 5
max = 7
```

## Environment Variables

Volato does not currently support environment variables for configuration. All settings must be in the TOML config file.

For sensitive values (API keys, tokens), ensure your config file has restricted permissions:

```bash
chmod 600 ~/.config/volato/config.toml
```
