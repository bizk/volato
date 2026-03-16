# Volato Setup Guide

This guide covers installing and configuring Volato on a Raspberry Pi (or any Linux system).

## Prerequisites

- Raspberry Pi 3 or newer (or any Linux system with ARM/AMD64)
- Go 1.21+ (for building from source)
- SQLite3
- Network access for API calls

## Installation

### Option 1: Build on the Pi

```bash
# Install Go (if not already installed)
wget https://go.dev/dl/go1.21.linux-armv6l.tar.gz
sudo tar -C /usr/local -xzf go1.21.linux-armv6l.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Clone and build
git clone https://github.com/override/volato.git
cd volato
go build -o volato ./cmd/volato
sudo mv volato /usr/local/bin/
```

### Option 2: Cross-Compile from Another Machine

```bash
# On your development machine (Linux/macOS)
git clone https://github.com/override/volato.git
cd volato

# Build for Raspberry Pi (ARM)
GOOS=linux GOARCH=arm GOARM=7 go build -o volato-arm ./cmd/volato

# Copy to Pi
scp volato-arm pi@raspberrypi:~/volato
ssh pi@raspberrypi 'sudo mv ~/volato /usr/local/bin/'
```

## Configuration

1. Create the config directory:
   ```bash
   mkdir -p ~/.config/volato
   ```

2. Copy and edit the example config:
   ```bash
   cp config.example.toml ~/.config/volato/config.toml
   nano ~/.config/volato/config.toml
   ```

3. Fill in your credentials:
   - Telegram bot token (from @BotFather)
   - Telegram chat ID (from @userinfobot)
   - At least one flight API key (Kiwi or Amadeus)

See [Configuration Reference](configuration.md) for all options.

## Database Initialization

Initialize the SQLite database:

```bash
volato migrate
```

This creates the database at `~/.local/share/volato/volato.db`.

To use a custom location:

```bash
volato migrate --db /path/to/volato.db
```

## Running Volato

### Manual Check

Run a one-time flight check:

```bash
volato check
```

### Cron Job Setup (Recommended)

Set up a daily check at 8 AM:

```bash
crontab -e
```

Add this line:

```cron
0 8 * * * /usr/local/bin/volato check >> /var/log/volato.log 2>&1
```

### Telegram Bot (Optional)

Run the Telegram bot for interactive commands:

```bash
volato bot
```

## Systemd Service Setup

For running the Telegram bot as a service:

1. Create the service file:

```bash
sudo nano /etc/systemd/system/volato-bot.service
```

2. Add the following content:

```ini
[Unit]
Description=Volato Flight Deals Bot
After=network.target

[Service]
Type=simple
User=pi
ExecStart=/usr/local/bin/volato bot
Restart=on-failure
RestartSec=10

[Install]
WantedBy=multi-user.target
```

3. Enable and start the service:

```bash
sudo systemctl daemon-reload
sudo systemctl enable volato-bot
sudo systemctl start volato-bot
```

4. Check status:

```bash
sudo systemctl status volato-bot
```

## Verifying Installation

1. Test the configuration:
   ```bash
   volato check --config ~/.config/volato/config.toml
   ```

2. Check the Telegram bot responds:
   - Send `/status` to your bot
   - Send `/help` to see available commands

## Troubleshooting

### "config file not found"

Ensure the config file exists at `~/.config/volato/config.toml` or specify the path:

```bash
volato check --config /path/to/config.toml
```

### "at least one API must be configured"

You need to configure either Kiwi or Amadeus API credentials. See [API Registration Guide](apis.md).

### "invalid chat_id"

The chat_id must be a numeric value. Get it from @userinfobot on Telegram.

### "database locked"

Only one instance of Volato can access the database at a time. Check for running processes:

```bash
ps aux | grep volato
```

### Bot not responding

1. Verify the bot token is correct
2. Ensure you started a conversation with the bot first
3. Check that your chat_id matches your Telegram account
4. View logs: `journalctl -u volato-bot -f`

### API rate limits

- Kiwi: 3,000 requests/month free tier
- Amadeus: 2,000 requests/month free tier

Consider reducing search frequency or number of destinations if hitting limits.

## Logs and Data

- Config: `~/.config/volato/config.toml`
- Database: `~/.local/share/volato/volato.db`
- Logs (cron): `/var/log/volato.log`
- Logs (systemd): `journalctl -u volato-bot`
