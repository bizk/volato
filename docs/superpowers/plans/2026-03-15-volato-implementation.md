# Volato Flight Deals Bot Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Go CLI that monitors flight prices from Argentina and sends Telegram deal alerts.

**Architecture:** Single binary with subcommands (check, bot, migrate). Queries Kiwi and Amadeus APIs, stores price history in SQLite, sends notifications via Telegram bot.

**Tech Stack:** Go 1.21+, SQLite, Cobra CLI, BurntSushi/toml, go-telegram-bot-api/v5

---

## File Structure

```
volato/
├── cmd/volato/main.go                 # CLI entry point with Cobra commands
├── internal/
│   ├── config/config.go               # TOML config parsing and validation
│   ├── config/config_test.go          # Config tests
│   ├── api/types.go                   # Flight, SearchRequest, FlightSearcher interface
│   ├── api/kiwi.go                    # Kiwi/Tequila API client
│   ├── api/kiwi_test.go               # Kiwi client tests
│   ├── api/amadeus.go                 # Amadeus API client with OAuth
│   ├── api/amadeus_test.go            # Amadeus client tests
│   ├── deals/detector.go              # Deal detection logic
│   ├── deals/detector_test.go         # Detector tests
│   ├── storage/sqlite.go              # SQLite operations
│   ├── storage/sqlite_test.go         # Storage tests
│   ├── telegram/notifier.go           # Send deal alerts
│   ├── telegram/notifier_test.go      # Notifier tests
│   ├── telegram/bot.go                # Bot command handler
│   ├── telegram/bot_test.go           # Bot tests
│   └── checker/checker.go             # Main orchestration
├── migrations/001_initial.sql         # Database schema
├── docs/
│   ├── setup.md                       # Raspberry Pi setup guide
│   ├── apis.md                        # API registration guide
│   └── configuration.md               # Config reference
├── config.example.toml                # Example config
├── go.mod
└── go.sum
```

---

## Chunk 1: Project Setup & Core Types

### Task 1.1: Initialize Go Module

**Files:**
- Create: `go.mod`

- [ ] **Step 1: Initialize Go module**

```bash
cd /home/override/code/personal/volato
go mod init github.com/override/volato
```

- [ ] **Step 2: Verify go.mod created**

Run: `cat go.mod`
Expected: Shows module path `github.com/override/volato`

- [ ] **Step 3: Commit**

```bash
git add go.mod
git commit -m "chore: initialize Go module"
```

---

### Task 1.2: Create Directory Structure

**Files:**
- Create: directories only

- [ ] **Step 1: Create all directories**

```bash
mkdir -p cmd/volato
mkdir -p internal/{config,api,deals,storage,telegram,checker}
mkdir -p migrations
mkdir -p docs
```

- [ ] **Step 2: Verify structure**

Run: `find . -type d | grep -v .git | sort`
Expected: Shows all directories

- [ ] **Step 3: Add .gitkeep files and commit**

```bash
touch internal/config/.gitkeep internal/api/.gitkeep internal/deals/.gitkeep
touch internal/storage/.gitkeep internal/telegram/.gitkeep internal/checker/.gitkeep
touch migrations/.gitkeep
git add .
git commit -m "chore: create project directory structure"
```

---

### Task 1.3: Define Core Types

**Files:**
- Create: `internal/api/types.go`

- [ ] **Step 1: Create types file**

```go
// internal/api/types.go
package api

import (
	"context"
	"time"
)

// Flight represents a flight offer from any API source.
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

// SearchRequest contains parameters for a flight search.
type SearchRequest struct {
	Origin      string
	Destination string
	DateFrom    time.Time
	DateTo      time.Time
	StayDaysMin int
	StayDaysMax int
	Currency    string
}

// FlightSearcher is the interface for flight API clients.
type FlightSearcher interface {
	Search(ctx context.Context, req SearchRequest) ([]Flight, error)
	Name() string
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/api/...`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add internal/api/types.go
git commit -m "feat: add core Flight and SearchRequest types"
```

---

### Task 1.4: Add Dependencies

**Files:**
- Modify: `go.mod`, `go.sum`

- [ ] **Step 1: Add Cobra for CLI**

```bash
go get github.com/spf13/cobra@latest
```

- [ ] **Step 2: Add TOML parser**

```bash
go get github.com/BurntSushi/toml@latest
```

- [ ] **Step 3: Add SQLite driver**

```bash
go get github.com/mattn/go-sqlite3@latest
```

- [ ] **Step 4: Add Telegram bot library**

```bash
go get github.com/go-telegram-bot-api/telegram-bot-api/v5@latest
```

- [ ] **Step 5: Tidy and commit**

```bash
go mod tidy
git add go.mod go.sum
git commit -m "chore: add project dependencies"
```

---

## Chunk 2: Configuration

### Task 2.1: Write Config Parsing Tests

**Files:**
- Create: `internal/config/config_test.go`

- [ ] **Step 1: Write failing test for config loading**

```go
// internal/config/config_test.go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_ValidConfig(t *testing.T) {
	// Create temp config file
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	content := `
[telegram]
bot_token = "123456:ABC"
chat_id = "999"

[apis.kiwi]
api_key = "kiwi_key"

[apis.amadeus]
client_id = "amadeus_id"
client_secret = "amadeus_secret"

[defaults]
origin = "EZE"
currency = "USD"

[alerts]
drop_threshold_percent = 20

[[searches]]
destination = "BCN"
months = [6, 7, 8]
max_price = 900

[searches.stay_days]
min = 10
max = 14
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Telegram.BotToken != "123456:ABC" {
		t.Errorf("BotToken = %q, want %q", cfg.Telegram.BotToken, "123456:ABC")
	}
	if cfg.Defaults.Origin != "EZE" {
		t.Errorf("Origin = %q, want %q", cfg.Defaults.Origin, "EZE")
	}
	if len(cfg.Searches) != 1 {
		t.Errorf("len(Searches) = %d, want 1", len(cfg.Searches))
	}
	if cfg.Searches[0].Destination != "BCN" {
		t.Errorf("Destination = %q, want %q", cfg.Searches[0].Destination, "BCN")
	}
	if cfg.Searches[0].StayDays.Min != 10 {
		t.Errorf("StayDays.Min = %d, want 10", cfg.Searches[0].StayDays.Min)
	}
}

func TestLoadConfig_MissingFile(t *testing.T) {
	_, err := Load("/nonexistent/config.toml")
	if err == nil {
		t.Error("Load() expected error for missing file")
	}
}

func TestLoadConfig_InvalidTOML(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")
	os.WriteFile(configPath, []byte("invalid toml [[["), 0644)

	_, err := Load(configPath)
	if err == nil {
		t.Error("Load() expected error for invalid TOML")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config/... -v`
Expected: FAIL - undefined: Load

- [ ] **Step 3: Commit failing test**

```bash
git add internal/config/config_test.go
git commit -m "test: add config loading tests (red)"
```

---

### Task 2.2: Implement Config Loading

**Files:**
- Create: `internal/config/config.go`

- [ ] **Step 1: Implement config structs and Load function**

```go
// internal/config/config.go
package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// Config is the root configuration structure.
type Config struct {
	Telegram TelegramConfig `toml:"telegram"`
	APIs     APIsConfig     `toml:"apis"`
	Defaults DefaultsConfig `toml:"defaults"`
	Alerts   AlertsConfig   `toml:"alerts"`
	Searches []SearchConfig `toml:"searches"`
}

// TelegramConfig holds Telegram bot settings.
type TelegramConfig struct {
	BotToken string `toml:"bot_token"`
	ChatID   string `toml:"chat_id"`
}

// APIsConfig holds API credentials.
type APIsConfig struct {
	Kiwi    KiwiConfig    `toml:"kiwi"`
	Amadeus AmadeusConfig `toml:"amadeus"`
}

// KiwiConfig holds Kiwi/Tequila API settings.
type KiwiConfig struct {
	APIKey string `toml:"api_key"`
}

// AmadeusConfig holds Amadeus API settings.
type AmadeusConfig struct {
	ClientID     string `toml:"client_id"`
	ClientSecret string `toml:"client_secret"`
}

// DefaultsConfig holds default search parameters.
type DefaultsConfig struct {
	Origin   string `toml:"origin"`
	Currency string `toml:"currency"`
}

// AlertsConfig holds alerting thresholds.
type AlertsConfig struct {
	DropThresholdPercent int `toml:"drop_threshold_percent"`
}

// SearchConfig defines a flight search to monitor.
type SearchConfig struct {
	Destination string       `toml:"destination"`
	Origin      string       `toml:"origin"` // Optional override
	Months      []int        `toml:"months"`
	StayDays    StayDays     `toml:"stay_days"`
	MaxPrice    float64      `toml:"max_price"`
}

// StayDays defines the trip duration range.
type StayDays struct {
	Min int `toml:"min"`
	Max int `toml:"max"`
}

// Load reads and parses a TOML config file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return &cfg, nil
}
```

- [ ] **Step 2: Run tests to verify they pass**

Run: `go test ./internal/config/... -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/config/config.go
git commit -m "feat: implement config loading from TOML"
```

---

### Task 2.3: Add Config Validation

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`

- [ ] **Step 1: Add validation test**

Add to `internal/config/config_test.go`:

```go
func TestConfig_Validate_MissingBotToken(t *testing.T) {
	cfg := &Config{
		Telegram: TelegramConfig{ChatID: "123"},
	}
	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() expected error for missing bot_token")
	}
}

func TestConfig_Validate_NoSearches(t *testing.T) {
	cfg := &Config{
		Telegram: TelegramConfig{BotToken: "token", ChatID: "123"},
		APIs: APIsConfig{
			Kiwi: KiwiConfig{APIKey: "key"},
		},
		Defaults: DefaultsConfig{Origin: "EZE", Currency: "USD"},
	}
	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() expected error for no searches")
	}
}

func TestConfig_Validate_Valid(t *testing.T) {
	cfg := &Config{
		Telegram: TelegramConfig{BotToken: "token", ChatID: "123"},
		APIs: APIsConfig{
			Kiwi: KiwiConfig{APIKey: "key"},
		},
		Defaults: DefaultsConfig{Origin: "EZE", Currency: "USD"},
		Alerts:   AlertsConfig{DropThresholdPercent: 20},
		Searches: []SearchConfig{
			{Destination: "BCN", Months: []int{6}, StayDays: StayDays{Min: 7, Max: 14}, MaxPrice: 900},
		},
	}
	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() unexpected error: %v", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/config/... -v`
Expected: FAIL - undefined: Validate

- [ ] **Step 3: Implement Validate method**

Add to `internal/config/config.go`:

```go
// Validate checks that required config fields are present.
func (c *Config) Validate() error {
	if c.Telegram.BotToken == "" {
		return fmt.Errorf("telegram.bot_token is required")
	}
	if c.Telegram.ChatID == "" {
		return fmt.Errorf("telegram.chat_id is required")
	}
	if c.APIs.Kiwi.APIKey == "" && c.APIs.Amadeus.ClientID == "" {
		return fmt.Errorf("at least one API (kiwi or amadeus) must be configured")
	}
	if c.Defaults.Origin == "" {
		return fmt.Errorf("defaults.origin is required")
	}
	if c.Defaults.Currency == "" {
		return fmt.Errorf("defaults.currency is required")
	}
	if len(c.Searches) == 0 {
		return fmt.Errorf("at least one [[searches]] entry is required")
	}
	for i, s := range c.Searches {
		if s.Destination == "" {
			return fmt.Errorf("searches[%d].destination is required", i)
		}
		if len(s.Months) == 0 {
			return fmt.Errorf("searches[%d].months is required", i)
		}
		if s.StayDays.Min <= 0 || s.StayDays.Max <= 0 {
			return fmt.Errorf("searches[%d].stay_days min/max must be positive", i)
		}
		if s.MaxPrice <= 0 {
			return fmt.Errorf("searches[%d].max_price must be positive", i)
		}
	}
	return nil
}

// EffectiveOrigin returns the origin for a search, using the default if not overridden.
func (c *Config) EffectiveOrigin(s SearchConfig) string {
	if s.Origin != "" {
		return s.Origin
	}
	return c.Defaults.Origin
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/config/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat: add config validation"
```

---

## Chunk 3: Storage Layer

### Task 3.1: Create Database Migration

**Files:**
- Create: `migrations/001_initial.sql`

- [ ] **Step 1: Create migration file**

```sql
-- migrations/001_initial.sql

-- Track price history for drop detection
CREATE TABLE IF NOT EXISTS price_history (
    id INTEGER PRIMARY KEY,
    origin TEXT NOT NULL,
    destination TEXT NOT NULL,
    departure_date TEXT NOT NULL,
    return_date TEXT NOT NULL,
    price REAL NOT NULL,
    currency TEXT NOT NULL,
    api_source TEXT NOT NULL,
    checked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Avoid sending duplicate alerts for the same deal
CREATE TABLE IF NOT EXISTS sent_deals (
    id INTEGER PRIMARY KEY,
    flight_hash TEXT UNIQUE NOT NULL,
    sent_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Track last check time for /status command
CREATE TABLE IF NOT EXISTS metadata (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

-- Index for efficient price history queries
CREATE INDEX IF NOT EXISTS idx_price_history_route
ON price_history(origin, destination, departure_date, return_date);

-- Index for cleanup queries
CREATE INDEX IF NOT EXISTS idx_price_history_checked_at
ON price_history(checked_at);

CREATE INDEX IF NOT EXISTS idx_sent_deals_sent_at
ON sent_deals(sent_at);
```

- [ ] **Step 2: Commit**

```bash
git add migrations/001_initial.sql
git commit -m "feat: add initial database migration"
```

---

### Task 3.2: Write Storage Tests

**Files:**
- Create: `internal/storage/sqlite_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/storage/sqlite_test.go
package storage

import (
	"path/filepath"
	"testing"
	"time"
)

func TestStore_SaveAndGetPriceHistory(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	entry := PriceEntry{
		Origin:        "EZE",
		Destination:   "BCN",
		DepartureDate: "2025-06-15",
		ReturnDate:    "2025-06-25",
		Price:         750.0,
		Currency:      "USD",
		APISource:     "kiwi",
	}

	if err := db.SavePrice(entry); err != nil {
		t.Fatalf("SavePrice() error = %v", err)
	}

	avg, err := db.GetAveragePrice("EZE", "BCN", "2025-06-15", "2025-06-25", 30)
	if err != nil {
		t.Fatalf("GetAveragePrice() error = %v", err)
	}
	if avg != 750.0 {
		t.Errorf("GetAveragePrice() = %v, want 750.0", avg)
	}
}

func TestStore_DealDeduplication(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	hash := "abc123"

	// First time should not be duplicate
	isDup, err := db.IsDealSent(hash)
	if err != nil {
		t.Fatalf("IsDealSent() error = %v", err)
	}
	if isDup {
		t.Error("IsDealSent() = true, want false for new deal")
	}

	// Mark as sent
	if err := db.MarkDealSent(hash); err != nil {
		t.Fatalf("MarkDealSent() error = %v", err)
	}

	// Now should be duplicate
	isDup, err = db.IsDealSent(hash)
	if err != nil {
		t.Fatalf("IsDealSent() error = %v", err)
	}
	if !isDup {
		t.Error("IsDealSent() = false, want true after marking sent")
	}
}

func TestStore_GetRecentDeals(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	// Save some price entries
	for i := 0; i < 15; i++ {
		db.SavePrice(PriceEntry{
			Origin:        "EZE",
			Destination:   "BCN",
			DepartureDate: "2025-06-15",
			ReturnDate:    "2025-06-25",
			Price:         float64(700 + i*10),
			Currency:      "USD",
			APISource:     "kiwi",
		})
	}

	deals, err := db.GetRecentPrices(10)
	if err != nil {
		t.Fatalf("GetRecentPrices() error = %v", err)
	}
	if len(deals) != 10 {
		t.Errorf("GetRecentPrices(10) returned %d, want 10", len(deals))
	}
}

func TestStore_Cleanup(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	// This test just verifies cleanup runs without error
	err := db.Cleanup(90, 7)
	if err != nil {
		t.Errorf("Cleanup() error = %v", err)
	}
}

func TestStore_Metadata(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	now := time.Now().Format(time.RFC3339)
	if err := db.SetMetadata("last_check", now); err != nil {
		t.Fatalf("SetMetadata() error = %v", err)
	}

	val, err := db.GetMetadata("last_check")
	if err != nil {
		t.Fatalf("GetMetadata() error = %v", err)
	}
	if val != now {
		t.Errorf("GetMetadata() = %q, want %q", val, now)
	}
}

func newTestDB(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return db
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/storage/... -v`
Expected: FAIL - undefined types

- [ ] **Step 3: Commit failing tests**

```bash
git add internal/storage/sqlite_test.go
git commit -m "test: add storage layer tests (red)"
```

---

### Task 3.3: Implement Storage Layer

**Files:**
- Create: `internal/storage/sqlite.go`

- [ ] **Step 1: Implement Store**

```go
// internal/storage/sqlite.go
package storage

import (
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed ../../../migrations/001_initial.sql
var migrationSQL string

// PriceEntry represents a price history record.
type PriceEntry struct {
	Origin        string
	Destination   string
	DepartureDate string
	ReturnDate    string
	Price         float64
	Currency      string
	APISource     string
	CheckedAt     string
}

// Store handles SQLite database operations.
type Store struct {
	db *sql.DB
}

// New creates a new Store, initializing the database if needed.
func New(dbPath string) (*Store, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating db directory: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Run migrations
	if _, err := db.Exec(migrationSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return &Store{db: db}, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// SavePrice stores a price entry in the history.
func (s *Store) SavePrice(e PriceEntry) error {
	_, err := s.db.Exec(`
		INSERT INTO price_history (origin, destination, departure_date, return_date, price, currency, api_source)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, e.Origin, e.Destination, e.DepartureDate, e.ReturnDate, e.Price, e.Currency, e.APISource)
	return err
}

// GetAveragePrice returns the average price for a route over the last N days.
// Returns 0 if no data available.
func (s *Store) GetAveragePrice(origin, destination, departureDate, returnDate string, days int) (float64, error) {
	var avg sql.NullFloat64
	err := s.db.QueryRow(`
		SELECT AVG(price) FROM price_history
		WHERE origin = ? AND destination = ?
		AND departure_date = ? AND return_date = ?
		AND checked_at >= datetime('now', '-' || ? || ' days')
	`, origin, destination, departureDate, returnDate, days).Scan(&avg)
	if err != nil {
		return 0, err
	}
	if !avg.Valid {
		return 0, nil
	}
	return avg.Float64, nil
}

// IsDealSent checks if a deal has already been sent.
func (s *Store) IsDealSent(hash string) (bool, error) {
	var count int
	err := s.db.QueryRow(`
		SELECT COUNT(*) FROM sent_deals
		WHERE flight_hash = ?
		AND sent_at >= datetime('now', '-1 day')
	`, hash).Scan(&count)
	return count > 0, err
}

// MarkDealSent records that a deal was sent.
func (s *Store) MarkDealSent(hash string) error {
	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO sent_deals (flight_hash, sent_at)
		VALUES (?, datetime('now'))
	`, hash)
	return err
}

// GetRecentPrices returns the N most recent price entries.
func (s *Store) GetRecentPrices(limit int) ([]PriceEntry, error) {
	rows, err := s.db.Query(`
		SELECT origin, destination, departure_date, return_date, price, currency, api_source, checked_at
		FROM price_history
		ORDER BY checked_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []PriceEntry
	for rows.Next() {
		var e PriceEntry
		if err := rows.Scan(&e.Origin, &e.Destination, &e.DepartureDate, &e.ReturnDate, &e.Price, &e.Currency, &e.APISource, &e.CheckedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// Cleanup removes old records.
func (s *Store) Cleanup(priceHistoryDays, sentDealsDays int) error {
	_, err := s.db.Exec(`DELETE FROM price_history WHERE checked_at < datetime('now', '-' || ? || ' days')`, priceHistoryDays)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`DELETE FROM sent_deals WHERE sent_at < datetime('now', '-' || ? || ' days')`, sentDealsDays)
	return err
}

// SetMetadata stores a key-value pair.
func (s *Store) SetMetadata(key, value string) error {
	_, err := s.db.Exec(`INSERT OR REPLACE INTO metadata (key, value) VALUES (?, ?)`, key, value)
	return err
}

// GetMetadata retrieves a value by key.
func (s *Store) GetMetadata(key string) (string, error) {
	var value string
	err := s.db.QueryRow(`SELECT value FROM metadata WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}
```

- [ ] **Step 2: Fix embed path - migrations need to be embedded correctly**

The embed directive needs to reference from the package location. Update the embed:

```go
// Remove the embed and load migrations differently
```

Actually, let me fix this - we need to pass migrations path or embed differently. Let's use a simpler approach:

```go
// internal/storage/sqlite.go
package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// Migration SQL embedded directly
const migrationSQL = `
CREATE TABLE IF NOT EXISTS price_history (
    id INTEGER PRIMARY KEY,
    origin TEXT NOT NULL,
    destination TEXT NOT NULL,
    departure_date TEXT NOT NULL,
    return_date TEXT NOT NULL,
    price REAL NOT NULL,
    currency TEXT NOT NULL,
    api_source TEXT NOT NULL,
    checked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sent_deals (
    id INTEGER PRIMARY KEY,
    flight_hash TEXT UNIQUE NOT NULL,
    sent_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS metadata (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_price_history_route
ON price_history(origin, destination, departure_date, return_date);

CREATE INDEX IF NOT EXISTS idx_price_history_checked_at
ON price_history(checked_at);

CREATE INDEX IF NOT EXISTS idx_sent_deals_sent_at
ON sent_deals(sent_at);
`

// PriceEntry represents a price history record.
type PriceEntry struct {
	Origin        string
	Destination   string
	DepartureDate string
	ReturnDate    string
	Price         float64
	Currency      string
	APISource     string
	CheckedAt     string
}

// Store handles SQLite database operations.
type Store struct {
	db *sql.DB
}

// New creates a new Store, initializing the database if needed.
func New(dbPath string) (*Store, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating db directory: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	if _, err := db.Exec(migrationSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return &Store{db: db}, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// SavePrice stores a price entry in the history.
func (s *Store) SavePrice(e PriceEntry) error {
	_, err := s.db.Exec(`
		INSERT INTO price_history (origin, destination, departure_date, return_date, price, currency, api_source)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, e.Origin, e.Destination, e.DepartureDate, e.ReturnDate, e.Price, e.Currency, e.APISource)
	return err
}

// GetAveragePrice returns the average price for a route over the last N days.
func (s *Store) GetAveragePrice(origin, destination, departureDate, returnDate string, days int) (float64, error) {
	var avg sql.NullFloat64
	err := s.db.QueryRow(`
		SELECT AVG(price) FROM price_history
		WHERE origin = ? AND destination = ?
		AND departure_date = ? AND return_date = ?
		AND checked_at >= datetime('now', '-' || ? || ' days')
	`, origin, destination, departureDate, returnDate, days).Scan(&avg)
	if err != nil {
		return 0, err
	}
	if !avg.Valid {
		return 0, nil
	}
	return avg.Float64, nil
}

// IsDealSent checks if a deal has already been sent within the last 24 hours.
func (s *Store) IsDealSent(hash string) (bool, error) {
	var count int
	err := s.db.QueryRow(`
		SELECT COUNT(*) FROM sent_deals
		WHERE flight_hash = ?
		AND sent_at >= datetime('now', '-1 day')
	`, hash).Scan(&count)
	return count > 0, err
}

// MarkDealSent records that a deal was sent.
func (s *Store) MarkDealSent(hash string) error {
	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO sent_deals (flight_hash, sent_at)
		VALUES (?, datetime('now'))
	`, hash)
	return err
}

// GetRecentPrices returns the N most recent price entries.
func (s *Store) GetRecentPrices(limit int) ([]PriceEntry, error) {
	rows, err := s.db.Query(`
		SELECT origin, destination, departure_date, return_date, price, currency, api_source, checked_at
		FROM price_history
		ORDER BY checked_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []PriceEntry
	for rows.Next() {
		var e PriceEntry
		if err := rows.Scan(&e.Origin, &e.Destination, &e.DepartureDate, &e.ReturnDate, &e.Price, &e.Currency, &e.APISource, &e.CheckedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// Cleanup removes old records.
func (s *Store) Cleanup(priceHistoryDays, sentDealsDays int) error {
	_, err := s.db.Exec(`DELETE FROM price_history WHERE checked_at < datetime('now', '-' || ? || ' days')`, priceHistoryDays)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`DELETE FROM sent_deals WHERE sent_at < datetime('now', '-' || ? || ' days')`, sentDealsDays)
	return err
}

// SetMetadata stores a key-value pair.
func (s *Store) SetMetadata(key, value string) error {
	_, err := s.db.Exec(`INSERT OR REPLACE INTO metadata (key, value) VALUES (?, ?)`, key, value)
	return err
}

// GetMetadata retrieves a value by key.
func (s *Store) GetMetadata(key string) (string, error) {
	var value string
	err := s.db.QueryRow(`SELECT value FROM metadata WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}
```

- [ ] **Step 3: Run tests to verify they pass**

Run: `go test ./internal/storage/... -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/storage/sqlite.go
git commit -m "feat: implement SQLite storage layer"
```

---

## Chunk 4: Kiwi API Client

### Task 4.1: Write Kiwi Client Tests

**Files:**
- Create: `internal/api/kiwi_test.go`

- [ ] **Step 1: Write failing tests with mock server**

```go
// internal/api/kiwi_test.go
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestKiwiClient_Search(t *testing.T) {
	// Mock Kiwi API response
	mockResp := map[string]interface{}{
		"data": []map[string]interface{}{
			{
				"flyFrom":    "EZE",
				"flyTo":      "BCN",
				"price":      750,
				"deep_link":  "https://kiwi.com/booking/123",
				"local_departure": "2025-06-15T10:00:00.000Z",
				"local_arrival":   "2025-06-16T08:00:00.000Z",
				"route": []map[string]interface{}{
					{"airline": "IB"},
				},
				"airlines": []string{"IB"},
			},
		},
		"currency": "USD",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify API key header
		if r.Header.Get("apikey") != "test-key" {
			t.Errorf("Missing or wrong apikey header")
		}
		json.NewEncoder(w).Encode(mockResp)
	}))
	defer server.Close()

	client := NewKiwiClient("test-key", server.URL)

	req := SearchRequest{
		Origin:      "EZE",
		Destination: "BCN",
		DateFrom:    time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		DateTo:      time.Date(2025, 6, 30, 0, 0, 0, 0, time.UTC),
		StayDaysMin: 10,
		StayDaysMax: 14,
		Currency:    "USD",
	}

	flights, err := client.Search(context.Background(), req)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(flights) != 1 {
		t.Fatalf("len(flights) = %d, want 1", len(flights))
	}

	f := flights[0]
	if f.Origin != "EZE" {
		t.Errorf("Origin = %q, want EZE", f.Origin)
	}
	if f.Destination != "BCN" {
		t.Errorf("Destination = %q, want BCN", f.Destination)
	}
	if f.Price != 750 {
		t.Errorf("Price = %v, want 750", f.Price)
	}
	if f.APISource != "kiwi" {
		t.Errorf("APISource = %q, want kiwi", f.APISource)
	}
}

func TestKiwiClient_Name(t *testing.T) {
	client := NewKiwiClient("key", "")
	if client.Name() != "kiwi" {
		t.Errorf("Name() = %q, want kiwi", client.Name())
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/api/... -v`
Expected: FAIL - undefined: NewKiwiClient

- [ ] **Step 3: Commit failing tests**

```bash
git add internal/api/kiwi_test.go
git commit -m "test: add Kiwi API client tests (red)"
```

---

### Task 4.2: Implement Kiwi Client

**Files:**
- Create: `internal/api/kiwi.go`

- [ ] **Step 1: Implement KiwiClient**

```go
// internal/api/kiwi.go
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const defaultKiwiBaseURL = "https://api.tequila.kiwi.com"

// KiwiClient implements FlightSearcher for Kiwi/Tequila API.
type KiwiClient struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// NewKiwiClient creates a new Kiwi API client.
func NewKiwiClient(apiKey, baseURL string) *KiwiClient {
	if baseURL == "" {
		baseURL = defaultKiwiBaseURL
	}
	return &KiwiClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

// Name returns the API name.
func (c *KiwiClient) Name() string {
	return "kiwi"
}

// kiwiResponse represents the Kiwi API response structure.
type kiwiResponse struct {
	Data     []kiwiFlightData `json:"data"`
	Currency string           `json:"currency"`
}

type kiwiFlightData struct {
	FlyFrom        string   `json:"flyFrom"`
	FlyTo          string   `json:"flyTo"`
	Price          float64  `json:"price"`
	DeepLink       string   `json:"deep_link"`
	LocalDeparture string   `json:"local_departure"`
	LocalArrival   string   `json:"local_arrival"`
	Airlines       []string `json:"airlines"`
	Route          []struct {
		Airline string `json:"airline"`
	} `json:"route"`
}

// Search queries the Kiwi API for flights.
func (c *KiwiClient) Search(ctx context.Context, req SearchRequest) ([]Flight, error) {
	u, err := url.Parse(c.baseURL + "/v2/search")
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("fly_from", req.Origin)
	q.Set("fly_to", req.Destination)
	q.Set("date_from", req.DateFrom.Format("02/01/2006"))
	q.Set("date_to", req.DateTo.Format("02/01/2006"))
	q.Set("nights_in_dst_from", fmt.Sprintf("%d", req.StayDaysMin))
	q.Set("nights_in_dst_to", fmt.Sprintf("%d", req.StayDaysMax))
	q.Set("curr", req.Currency)
	q.Set("flight_type", "round")
	q.Set("one_for_city", "0")
	q.Set("partner", "picky")
	u.RawQuery = q.Encode()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("apikey", c.apiKey)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("kiwi request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("kiwi returned status %d", resp.StatusCode)
	}

	var kiwiResp kiwiResponse
	if err := json.NewDecoder(resp.Body).Decode(&kiwiResp); err != nil {
		return nil, fmt.Errorf("decoding kiwi response: %w", err)
	}

	flights := make([]Flight, 0, len(kiwiResp.Data))
	for _, d := range kiwiResp.Data {
		depTime, _ := time.Parse(time.RFC3339, d.LocalDeparture)

		// Calculate return date from route (simplified - last segment)
		var retTime time.Time
		if len(d.Route) > 0 {
			retTime = depTime.AddDate(0, 0, req.StayDaysMin) // Approximation
		}

		airline := ""
		if len(d.Airlines) > 0 {
			airline = d.Airlines[0]
		}

		flights = append(flights, Flight{
			Origin:        d.FlyFrom,
			Destination:   d.FlyTo,
			DepartureDate: depTime,
			ReturnDate:    retTime,
			Price:         d.Price,
			Currency:      kiwiResp.Currency,
			Airline:       airline,
			Stops:         len(d.Route) - 1,
			BookingLink:   d.DeepLink,
			APISource:     "kiwi",
		})
	}

	return flights, nil
}
```

- [ ] **Step 2: Run tests to verify they pass**

Run: `go test ./internal/api/... -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/api/kiwi.go
git commit -m "feat: implement Kiwi/Tequila API client"
```

---

## Chunk 5: Amadeus API Client

### Task 5.1: Write Amadeus Client Tests

**Files:**
- Create: `internal/api/amadeus_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/api/amadeus_test.go
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAmadeusClient_Search(t *testing.T) {
	tokenResp := map[string]interface{}{
		"access_token": "test-token",
		"token_type":   "Bearer",
		"expires_in":   1799,
	}

	flightResp := map[string]interface{}{
		"data": []map[string]interface{}{
			{
				"price": map[string]interface{}{
					"total":    "850.00",
					"currency": "USD",
				},
				"itineraries": []map[string]interface{}{
					{
						"segments": []map[string]interface{}{
							{
								"departure": map[string]string{
									"iataCode": "EZE",
									"at":       "2025-06-15T10:00:00",
								},
								"arrival": map[string]string{
									"iataCode": "BCN",
									"at":       "2025-06-16T08:00:00",
								},
								"carrierCode": "IB",
							},
						},
					},
					{
						"segments": []map[string]interface{}{
							{
								"departure": map[string]string{
									"iataCode": "BCN",
									"at":       "2025-06-25T10:00:00",
								},
								"arrival": map[string]string{
									"iataCode": "EZE",
									"at":       "2025-06-26T08:00:00",
								},
								"carrierCode": "IB",
							},
						},
					},
				},
			},
		},
	}

	reqCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqCount++
		if r.URL.Path == "/v1/security/oauth2/token" {
			json.NewEncoder(w).Encode(tokenResp)
			return
		}
		if r.URL.Path == "/v2/shopping/flight-offers" {
			// Verify auth header
			if r.Header.Get("Authorization") != "Bearer test-token" {
				t.Errorf("Missing or wrong Authorization header")
			}
			json.NewEncoder(w).Encode(flightResp)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := NewAmadeusClient("client-id", "client-secret", server.URL)

	req := SearchRequest{
		Origin:      "EZE",
		Destination: "BCN",
		DateFrom:    time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
		DateTo:      time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
		StayDaysMin: 10,
		StayDaysMax: 10,
		Currency:    "USD",
	}

	flights, err := client.Search(context.Background(), req)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(flights) != 1 {
		t.Fatalf("len(flights) = %d, want 1", len(flights))
	}

	f := flights[0]
	if f.Price != 850.00 {
		t.Errorf("Price = %v, want 850", f.Price)
	}
	if f.APISource != "amadeus" {
		t.Errorf("APISource = %q, want amadeus", f.APISource)
	}
}

func TestAmadeusClient_Name(t *testing.T) {
	client := NewAmadeusClient("id", "secret", "")
	if client.Name() != "amadeus" {
		t.Errorf("Name() = %q, want amadeus", client.Name())
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/api/... -v`
Expected: FAIL - undefined: NewAmadeusClient

- [ ] **Step 3: Commit failing tests**

```bash
git add internal/api/amadeus_test.go
git commit -m "test: add Amadeus API client tests (red)"
```

---

### Task 5.2: Implement Amadeus Client

**Files:**
- Create: `internal/api/amadeus.go`

- [ ] **Step 1: Implement AmadeusClient**

```go
// internal/api/amadeus.go
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const defaultAmadeusBaseURL = "https://api.amadeus.com"

// AmadeusClient implements FlightSearcher for Amadeus API.
type AmadeusClient struct {
	clientID     string
	clientSecret string
	baseURL      string
	client       *http.Client

	mu          sync.Mutex
	accessToken string
	tokenExpiry time.Time
}

// NewAmadeusClient creates a new Amadeus API client.
func NewAmadeusClient(clientID, clientSecret, baseURL string) *AmadeusClient {
	if baseURL == "" {
		baseURL = defaultAmadeusBaseURL
	}
	return &AmadeusClient{
		clientID:     clientID,
		clientSecret: clientSecret,
		baseURL:      baseURL,
		client:       &http.Client{Timeout: 30 * time.Second},
	}
}

// Name returns the API name.
func (c *AmadeusClient) Name() string {
	return "amadeus"
}

type amadeusTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

type amadeusFlightResponse struct {
	Data []amadeusFlightOffer `json:"data"`
}

type amadeusFlightOffer struct {
	Price       amadeusPrice       `json:"price"`
	Itineraries []amadeusItinerary `json:"itineraries"`
}

type amadeusPrice struct {
	Total    string `json:"total"`
	Currency string `json:"currency"`
}

type amadeusItinerary struct {
	Segments []amadeusSegment `json:"segments"`
}

type amadeusSegment struct {
	Departure   amadeusLocation `json:"departure"`
	Arrival     amadeusLocation `json:"arrival"`
	CarrierCode string          `json:"carrierCode"`
}

type amadeusLocation struct {
	IataCode string `json:"iataCode"`
	At       string `json:"at"`
}

func (c *AmadeusClient) getToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Return cached token if still valid
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry) {
		return c.accessToken, nil
	}

	// Request new token
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", c.clientID)
	data.Set("client_secret", c.clientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/v1/security/oauth2/token",
		strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token request returned status %d", resp.StatusCode)
	}

	var tokenResp amadeusTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}

	c.accessToken = tokenResp.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn-60) * time.Second)

	return c.accessToken, nil
}

// Search queries the Amadeus API for flights.
func (c *AmadeusClient) Search(ctx context.Context, req SearchRequest) ([]Flight, error) {
	token, err := c.getToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting token: %w", err)
	}

	u, err := url.Parse(c.baseURL + "/v2/shopping/flight-offers")
	if err != nil {
		return nil, err
	}

	// Amadeus requires specific departure date, not range
	// We'll search for the start date
	departureDate := req.DateFrom.Format("2006-01-02")
	returnDate := req.DateFrom.AddDate(0, 0, req.StayDaysMin).Format("2006-01-02")

	q := u.Query()
	q.Set("originLocationCode", req.Origin)
	q.Set("destinationLocationCode", req.Destination)
	q.Set("departureDate", departureDate)
	q.Set("returnDate", returnDate)
	q.Set("adults", "1")
	q.Set("currencyCode", req.Currency)
	q.Set("max", "50")
	u.RawQuery = q.Encode()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("amadeus request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("amadeus returned status %d", resp.StatusCode)
	}

	var amadeusResp amadeusFlightResponse
	if err := json.NewDecoder(resp.Body).Decode(&amadeusResp); err != nil {
		return nil, fmt.Errorf("decoding amadeus response: %w", err)
	}

	flights := make([]Flight, 0, len(amadeusResp.Data))
	for _, offer := range amadeusResp.Data {
		if len(offer.Itineraries) < 2 {
			continue // Need outbound and return
		}

		outbound := offer.Itineraries[0]
		inbound := offer.Itineraries[1]

		if len(outbound.Segments) == 0 || len(inbound.Segments) == 0 {
			continue
		}

		firstSeg := outbound.Segments[0]
		lastReturnSeg := inbound.Segments[len(inbound.Segments)-1]

		depTime, _ := time.Parse("2006-01-02T15:04:05", firstSeg.Departure.At)
		retTime, _ := time.Parse("2006-01-02T15:04:05", lastReturnSeg.Arrival.At)

		price, _ := strconv.ParseFloat(offer.Price.Total, 64)

		// Count stops (segments - 1 for each direction)
		stops := (len(outbound.Segments) - 1) + (len(inbound.Segments) - 1)

		flights = append(flights, Flight{
			Origin:        firstSeg.Departure.IataCode,
			Destination:   firstSeg.Arrival.IataCode,
			DepartureDate: depTime,
			ReturnDate:    retTime,
			Price:         price,
			Currency:      offer.Price.Currency,
			Airline:       firstSeg.CarrierCode,
			Stops:         stops,
			BookingLink:   "", // Amadeus doesn't provide direct booking links in basic API
			APISource:     "amadeus",
		})
	}

	return flights, nil
}
```

- [ ] **Step 2: Run tests to verify they pass**

Run: `go test ./internal/api/... -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/api/amadeus.go
git commit -m "feat: implement Amadeus API client with OAuth"
```

---

## Chunk 6: Deal Detection

### Task 6.1: Write Deal Detection Tests

**Files:**
- Create: `internal/deals/detector_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/deals/detector_test.go
package deals

import (
	"testing"
	"time"

	"github.com/override/volato/internal/api"
)

type mockStore struct {
	avgPrice  float64
	sentDeals map[string]bool
}

func (m *mockStore) GetAveragePrice(origin, dest, dep, ret string, days int) (float64, error) {
	return m.avgPrice, nil
}

func (m *mockStore) IsDealSent(hash string) (bool, error) {
	return m.sentDeals[hash], nil
}

func (m *mockStore) MarkDealSent(hash string) error {
	if m.sentDeals == nil {
		m.sentDeals = make(map[string]bool)
	}
	m.sentDeals[hash] = true
	return nil
}

func (m *mockStore) SavePrice(origin, dest, dep, ret string, price float64, currency, source string) error {
	return nil
}

func TestDetector_BelowMaxPrice(t *testing.T) {
	store := &mockStore{avgPrice: 0}
	d := New(store, 20)

	flight := api.Flight{
		Origin:        "EZE",
		Destination:   "BCN",
		DepartureDate: time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
		ReturnDate:    time.Date(2025, 6, 25, 0, 0, 0, 0, time.UTC),
		Price:         750,
		Currency:      "USD",
	}

	result := d.Evaluate(flight, 900)
	if !result.IsDeal {
		t.Error("Expected deal when price below max")
	}
	if result.Reason != ReasonBelowThreshold {
		t.Errorf("Reason = %v, want %v", result.Reason, ReasonBelowThreshold)
	}
}

func TestDetector_AboveMaxPrice(t *testing.T) {
	store := &mockStore{avgPrice: 0}
	d := New(store, 20)

	flight := api.Flight{
		Price: 950,
	}

	result := d.Evaluate(flight, 900)
	if result.IsDeal {
		t.Error("Expected no deal when price above max")
	}
}

func TestDetector_PriceDrop(t *testing.T) {
	store := &mockStore{avgPrice: 1000}
	d := New(store, 20)

	flight := api.Flight{
		Origin:        "EZE",
		Destination:   "BCN",
		DepartureDate: time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
		ReturnDate:    time.Date(2025, 6, 25, 0, 0, 0, 0, time.UTC),
		Price:         750, // 25% drop from 1000
		Currency:      "USD",
	}

	result := d.Evaluate(flight, 1500) // max price higher than current
	if !result.IsDeal {
		t.Error("Expected deal on significant price drop")
	}
	if result.Reason != ReasonPriceDrop {
		t.Errorf("Reason = %v, want %v", result.Reason, ReasonPriceDrop)
	}
}

func TestDetector_AlreadySent(t *testing.T) {
	store := &mockStore{avgPrice: 0, sentDeals: make(map[string]bool)}
	d := New(store, 20)

	flight := api.Flight{
		Origin:        "EZE",
		Destination:   "BCN",
		DepartureDate: time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
		ReturnDate:    time.Date(2025, 6, 25, 0, 0, 0, 0, time.UTC),
		Price:         750,
	}

	// First evaluation - should be deal
	result1 := d.Evaluate(flight, 900)
	if !result1.IsDeal {
		t.Error("First evaluation should be a deal")
	}

	// Mark as sent
	d.MarkSent(flight)

	// Second evaluation - should not be deal (already sent)
	result2 := d.Evaluate(flight, 900)
	if result2.IsDeal {
		t.Error("Second evaluation should not be a deal (already sent)")
	}
}

func TestFlightHash(t *testing.T) {
	f1 := api.Flight{
		Origin:        "EZE",
		Destination:   "BCN",
		DepartureDate: time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
		ReturnDate:    time.Date(2025, 6, 25, 0, 0, 0, 0, time.UTC),
		Price:         753, // rounds to 750
	}
	f2 := api.Flight{
		Origin:        "EZE",
		Destination:   "BCN",
		DepartureDate: time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
		ReturnDate:    time.Date(2025, 6, 25, 0, 0, 0, 0, time.UTC),
		Price:         757, // also rounds to 760
	}

	h1 := FlightHash(f1)
	h2 := FlightHash(f2)

	if h1 == "" {
		t.Error("Hash should not be empty")
	}
	// Prices within $10 bucket should NOT have same hash
	// 753 -> 750, 757 -> 760, so different buckets
	if h1 == h2 {
		t.Error("Different price buckets should have different hashes")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/deals/... -v`
Expected: FAIL - undefined types

- [ ] **Step 3: Commit failing tests**

```bash
git add internal/deals/detector_test.go
git commit -m "test: add deal detection tests (red)"
```

---

### Task 6.2: Implement Deal Detector

**Files:**
- Create: `internal/deals/detector.go`

- [ ] **Step 1: Implement Detector**

```go
// internal/deals/detector.go
package deals

import (
	"crypto/sha256"
	"fmt"
	"math"

	"github.com/override/volato/internal/api"
)

// DealReason indicates why a flight is considered a deal.
type DealReason string

const (
	ReasonBelowThreshold DealReason = "below_threshold"
	ReasonPriceDrop      DealReason = "price_drop"
	ReasonNoDeal         DealReason = ""
)

// EvaluationResult contains the deal evaluation outcome.
type EvaluationResult struct {
	IsDeal       bool
	Reason       DealReason
	DropPercent  float64
	AveragePrice float64
}

// Store defines the storage operations needed by the detector.
type Store interface {
	GetAveragePrice(origin, dest, dep, ret string, days int) (float64, error)
	IsDealSent(hash string) (bool, error)
	MarkDealSent(hash string) error
	SavePrice(origin, dest, dep, ret string, price float64, currency, source string) error
}

// Detector evaluates flights to determine if they are deals.
type Detector struct {
	store              Store
	dropThresholdPct   int
	historyDays        int
}

// New creates a new deal detector.
func New(store Store, dropThresholdPct int) *Detector {
	return &Detector{
		store:            store,
		dropThresholdPct: dropThresholdPct,
		historyDays:      30,
	}
}

// Evaluate checks if a flight is a deal.
func (d *Detector) Evaluate(f api.Flight, maxPrice float64) EvaluationResult {
	depDate := f.DepartureDate.Format("2006-01-02")
	retDate := f.ReturnDate.Format("2006-01-02")

	// Check if already sent
	hash := FlightHash(f)
	sent, err := d.store.IsDealSent(hash)
	if err == nil && sent {
		return EvaluationResult{IsDeal: false, Reason: ReasonNoDeal}
	}

	// Check price threshold
	if f.Price <= maxPrice {
		return EvaluationResult{
			IsDeal: true,
			Reason: ReasonBelowThreshold,
		}
	}

	// Check for price drop
	avgPrice, err := d.store.GetAveragePrice(f.Origin, f.Destination, depDate, retDate, d.historyDays)
	if err != nil || avgPrice == 0 {
		return EvaluationResult{IsDeal: false, Reason: ReasonNoDeal}
	}

	dropPct := ((avgPrice - f.Price) / avgPrice) * 100
	if dropPct >= float64(d.dropThresholdPct) {
		return EvaluationResult{
			IsDeal:       true,
			Reason:       ReasonPriceDrop,
			DropPercent:  dropPct,
			AveragePrice: avgPrice,
		}
	}

	return EvaluationResult{IsDeal: false, Reason: ReasonNoDeal}
}

// MarkSent marks a flight deal as sent.
func (d *Detector) MarkSent(f api.Flight) error {
	return d.store.MarkDealSent(FlightHash(f))
}

// SavePrice stores a flight price in history.
func (d *Detector) SavePrice(f api.Flight) error {
	depDate := f.DepartureDate.Format("2006-01-02")
	retDate := f.ReturnDate.Format("2006-01-02")
	return d.store.SavePrice(f.Origin, f.Destination, depDate, retDate, f.Price, f.Currency, f.APISource)
}

// FlightHash generates a hash for deduplication.
// Price is bucketed to nearest $10 to avoid alerting on minor fluctuations.
func FlightHash(f api.Flight) string {
	priceBucket := int(math.Round(f.Price/10) * 10)
	data := fmt.Sprintf("%s|%s|%s|%s|%d",
		f.Origin,
		f.Destination,
		f.DepartureDate.Format("2006-01-02"),
		f.ReturnDate.Format("2006-01-02"),
		priceBucket,
	)
	h := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", h[:8]) // First 8 bytes is enough
}
```

- [ ] **Step 2: Run tests to verify they pass**

Run: `go test ./internal/deals/... -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/deals/detector.go
git commit -m "feat: implement deal detection with threshold and price drop"
```

---

## Chunk 7: Telegram Integration

### Task 7.1: Write Telegram Notifier Tests

**Files:**
- Create: `internal/telegram/notifier_test.go`

- [ ] **Step 1: Write tests**

```go
// internal/telegram/notifier_test.go
package telegram

import (
	"testing"
	"time"

	"github.com/override/volato/internal/api"
)

func TestFormatDealMessage(t *testing.T) {
	flight := api.Flight{
		Origin:        "EZE",
		Destination:   "BCN",
		DepartureDate: time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
		ReturnDate:    time.Date(2025, 6, 25, 0, 0, 0, 0, time.UTC),
		Price:         745,
		Currency:      "USD",
		BookingLink:   "https://kiwi.com/booking/123",
	}

	msg := FormatDealMessage(flight)

	// Check key parts are present
	if !contains(msg, "EZE") {
		t.Error("Message should contain origin")
	}
	if !contains(msg, "BCN") {
		t.Error("Message should contain destination")
	}
	if !contains(msg, "745") {
		t.Error("Message should contain price")
	}
	if !contains(msg, "USD") {
		t.Error("Message should contain currency")
	}
	if !contains(msg, "Jun 15") {
		t.Error("Message should contain departure date")
	}
	if !contains(msg, "10 days") {
		t.Error("Message should contain duration")
	}
	if !contains(msg, "https://kiwi.com") {
		t.Error("Message should contain booking link")
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		(s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || contains(s[1:], substr)))
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/telegram/... -v`
Expected: FAIL - undefined: FormatDealMessage

- [ ] **Step 3: Commit failing tests**

```bash
git add internal/telegram/notifier_test.go
git commit -m "test: add Telegram notifier tests (red)"
```

---

### Task 7.2: Implement Telegram Notifier

**Files:**
- Create: `internal/telegram/notifier.go`

- [ ] **Step 1: Implement Notifier**

```go
// internal/telegram/notifier.go
package telegram

import (
	"fmt"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/override/volato/internal/api"
)

// Notifier sends messages via Telegram.
type Notifier struct {
	bot    *tgbotapi.BotAPI
	chatID int64
}

// NewNotifier creates a new Telegram notifier.
func NewNotifier(token string, chatID int64) (*Notifier, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("creating bot: %w", err)
	}
	return &Notifier{bot: bot, chatID: chatID}, nil
}

// SendDeal sends a deal notification.
func (n *Notifier) SendDeal(flight api.Flight) error {
	msg := tgbotapi.NewMessage(n.chatID, FormatDealMessage(flight))
	msg.ParseMode = "HTML"
	msg.DisableWebPagePreview = true

	_, err := n.bot.Send(msg)
	return err
}

// SendText sends a plain text message.
func (n *Notifier) SendText(text string) error {
	msg := tgbotapi.NewMessage(n.chatID, text)
	_, err := n.bot.Send(msg)
	return err
}

// FormatDealMessage formats a flight as a deal notification.
func FormatDealMessage(f api.Flight) string {
	duration := int(f.ReturnDate.Sub(f.DepartureDate).Hours() / 24)

	var sb strings.Builder
	sb.WriteString("✈️ <b>Deal Found!</b>\n\n")
	sb.WriteString(fmt.Sprintf("🛫 %s → %s\n", f.Origin, f.Destination))
	sb.WriteString(fmt.Sprintf("📅 %s - %s (%d days)\n",
		f.DepartureDate.Format("Jan 2"),
		f.ReturnDate.Format("Jan 2, 2006"),
		duration))
	sb.WriteString(fmt.Sprintf("💰 $%.0f %s\n", f.Price, f.Currency))

	if f.BookingLink != "" {
		sb.WriteString(fmt.Sprintf("\n🔗 <a href=\"%s\">Book now</a>", f.BookingLink))
	}

	return sb.String()
}

// FormatPriceDropMessage adds drop info to the deal message.
func FormatPriceDropMessage(f api.Flight, dropPct, avgPrice float64) string {
	base := FormatDealMessage(f)
	dropInfo := fmt.Sprintf("\n\n📉 <i>%.0f%% below average ($%.0f)</i>", dropPct, avgPrice)
	return base + dropInfo
}
```

- [ ] **Step 2: Fix test helper**

Update `internal/telegram/notifier_test.go` to use strings.Contains:

```go
// internal/telegram/notifier_test.go
package telegram

import (
	"strings"
	"testing"
	"time"

	"github.com/override/volato/internal/api"
)

func TestFormatDealMessage(t *testing.T) {
	flight := api.Flight{
		Origin:        "EZE",
		Destination:   "BCN",
		DepartureDate: time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
		ReturnDate:    time.Date(2025, 6, 25, 0, 0, 0, 0, time.UTC),
		Price:         745,
		Currency:      "USD",
		BookingLink:   "https://kiwi.com/booking/123",
	}

	msg := FormatDealMessage(flight)

	checks := []string{"EZE", "BCN", "745", "USD", "Jun 15", "10 days", "kiwi.com"}
	for _, check := range checks {
		if !strings.Contains(msg, check) {
			t.Errorf("Message should contain %q, got: %s", check, msg)
		}
	}
}

func TestFormatPriceDropMessage(t *testing.T) {
	flight := api.Flight{
		Origin:        "EZE",
		Destination:   "BCN",
		DepartureDate: time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
		ReturnDate:    time.Date(2025, 6, 25, 0, 0, 0, 0, time.UTC),
		Price:         750,
		Currency:      "USD",
	}

	msg := FormatPriceDropMessage(flight, 25, 1000)

	if !strings.Contains(msg, "25%") {
		t.Error("Should contain drop percentage")
	}
	if !strings.Contains(msg, "1000") {
		t.Error("Should contain average price")
	}
}
```

- [ ] **Step 3: Run tests to verify they pass**

Run: `go test ./internal/telegram/... -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/telegram/notifier.go internal/telegram/notifier_test.go
git commit -m "feat: implement Telegram notifier"
```

---

### Task 7.3: Implement Telegram Bot Commands

**Files:**
- Create: `internal/telegram/bot.go`

- [ ] **Step 1: Implement Bot**

```go
// internal/telegram/bot.go
package telegram

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// CheckFunc is called when /check command is received.
type CheckFunc func(ctx context.Context) error

// StatusInfo contains bot status information.
type StatusInfo struct {
	LastCheck   time.Time
	DealsFound  int
	NextCheck   string
}

// Bot handles Telegram bot commands.
type Bot struct {
	bot        *tgbotapi.BotAPI
	chatID     int64
	checkFunc  CheckFunc
	statusFunc func() StatusInfo
	dealsFunc  func() []string

	mu            sync.Mutex
	lastCheckTime time.Time
}

// NewBot creates a new Telegram bot handler.
func NewBot(token string, chatID int64) (*Bot, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("creating bot: %w", err)
	}
	return &Bot{bot: bot, chatID: chatID}, nil
}

// SetCheckFunc sets the function to call for /check command.
func (b *Bot) SetCheckFunc(f CheckFunc) {
	b.checkFunc = f
}

// SetStatusFunc sets the function to call for /status command.
func (b *Bot) SetStatusFunc(f func() StatusInfo) {
	b.statusFunc = f
}

// SetDealsFunc sets the function to call for /deals command.
func (b *Bot) SetDealsFunc(f func() []string) {
	b.dealsFunc = f
}

// Run starts the bot and listens for commands.
func (b *Bot) Run(ctx context.Context) error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.bot.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case update := <-updates:
			if update.Message == nil || !update.Message.IsCommand() {
				continue
			}

			// Only respond to configured chat
			if update.Message.Chat.ID != b.chatID {
				continue
			}

			switch update.Message.Command() {
			case "status":
				b.handleStatus(update.Message.Chat.ID)
			case "check":
				b.handleCheck(ctx, update.Message.Chat.ID)
			case "deals":
				b.handleDeals(update.Message.Chat.ID)
			case "help":
				b.handleHelp(update.Message.Chat.ID)
			}
		}
	}
}

func (b *Bot) handleStatus(chatID int64) {
	var text string
	if b.statusFunc != nil {
		info := b.statusFunc()
		text = fmt.Sprintf("📊 <b>Status</b>\n\n"+
			"Last check: %s\n"+
			"Deals found: %d\n"+
			"Next check: %s",
			info.LastCheck.Format("Jan 2, 15:04"),
			info.DealsFound,
			info.NextCheck)
	} else {
		text = "Status information unavailable"
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	b.bot.Send(msg)
}

func (b *Bot) handleCheck(ctx context.Context, chatID int64) {
	// Rate limit: 1 hour cooldown
	b.mu.Lock()
	if time.Since(b.lastCheckTime) < time.Hour {
		remaining := time.Hour - time.Since(b.lastCheckTime)
		b.mu.Unlock()
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("⏳ Please wait %d minutes before checking again", int(remaining.Minutes())))
		b.bot.Send(msg)
		return
	}
	b.lastCheckTime = time.Now()
	b.mu.Unlock()

	msg := tgbotapi.NewMessage(chatID, "🔍 Starting flight check...")
	b.bot.Send(msg)

	if b.checkFunc != nil {
		if err := b.checkFunc(ctx); err != nil {
			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ Check failed: %v", err))
			b.bot.Send(msg)
			return
		}
	}

	msg = tgbotapi.NewMessage(chatID, "✅ Check complete!")
	b.bot.Send(msg)
}

func (b *Bot) handleDeals(chatID int64) {
	var text string
	if b.dealsFunc != nil {
		deals := b.dealsFunc()
		if len(deals) == 0 {
			text = "No recent deals found"
		} else {
			text = "📋 <b>Recent Deals</b>\n\n" + strings.Join(deals, "\n\n")
		}
	} else {
		text = "Deals information unavailable"
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	b.bot.Send(msg)
}

func (b *Bot) handleHelp(chatID int64) {
	text := `🤖 <b>Volato Bot Commands</b>

/status - Show bot status and last check time
/check - Trigger a manual flight check (1hr cooldown)
/deals - Show recent deals found
/help - Show this help message`

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	b.bot.Send(msg)
}
```

- [ ] **Step 2: Run all telegram tests**

Run: `go test ./internal/telegram/... -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/telegram/bot.go
git commit -m "feat: implement Telegram bot commands"
```

---

## Chunk 8: CLI & Checker Orchestration

### Task 8.1: Implement Checker

**Files:**
- Create: `internal/checker/checker.go`

- [ ] **Step 1: Implement Checker**

```go
// internal/checker/checker.go
package checker

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/override/volato/internal/api"
	"github.com/override/volato/internal/config"
	"github.com/override/volato/internal/deals"
	"github.com/override/volato/internal/storage"
	"github.com/override/volato/internal/telegram"
)

// Checker orchestrates the flight checking process.
type Checker struct {
	cfg      *config.Config
	store    *storage.Store
	detector *deals.Detector
	notifier *telegram.Notifier
	apis     []api.FlightSearcher
}

// New creates a new Checker.
func New(cfg *config.Config, store *storage.Store, notifier *telegram.Notifier) *Checker {
	// Create detector with store adapter
	storeAdapter := &storeAdapter{store: store}
	detector := deals.New(storeAdapter, cfg.Alerts.DropThresholdPercent)

	// Create API clients
	var apis []api.FlightSearcher
	if cfg.APIs.Kiwi.APIKey != "" {
		apis = append(apis, api.NewKiwiClient(cfg.APIs.Kiwi.APIKey, ""))
	}
	if cfg.APIs.Amadeus.ClientID != "" {
		apis = append(apis, api.NewAmadeusClient(
			cfg.APIs.Amadeus.ClientID,
			cfg.APIs.Amadeus.ClientSecret,
			"",
		))
	}

	return &Checker{
		cfg:      cfg,
		store:    store,
		detector: detector,
		notifier: notifier,
		apis:     apis,
	}
}

// Run executes the flight check.
func (c *Checker) Run(ctx context.Context) error {
	log.Println("Starting flight check...")

	// Cleanup old data
	if err := c.store.Cleanup(90, 7); err != nil {
		log.Printf("Warning: cleanup failed: %v", err)
	}

	dealsFound := 0

	for _, search := range c.cfg.Searches {
		origin := c.cfg.EffectiveOrigin(search)

		// Generate date ranges for configured months
		dateRanges := c.generateDateRanges(search.Months)

		for _, dr := range dateRanges {
			req := api.SearchRequest{
				Origin:      origin,
				Destination: search.Destination,
				DateFrom:    dr.from,
				DateTo:      dr.to,
				StayDaysMin: search.StayDays.Min,
				StayDaysMax: search.StayDays.Max,
				Currency:    c.cfg.Defaults.Currency,
			}

			// Query all APIs
			var allFlights []api.Flight
			for _, apiClient := range c.apis {
				flights, err := apiClient.Search(ctx, req)
				if err != nil {
					log.Printf("Warning: %s API error for %s->%s: %v",
						apiClient.Name(), origin, search.Destination, err)
					continue
				}
				allFlights = append(allFlights, flights...)
			}

			// Deduplicate and process
			flights := c.deduplicateFlights(allFlights)

			for _, f := range flights {
				// Save price to history
				c.detector.SavePrice(f)

				// Evaluate if it's a deal
				result := c.detector.Evaluate(f, search.MaxPrice)
				if !result.IsDeal {
					continue
				}

				// Send notification
				var err error
				if result.Reason == deals.ReasonPriceDrop {
					msg := telegram.FormatPriceDropMessage(f, result.DropPercent, result.AveragePrice)
					err = c.notifier.SendText(msg)
				} else {
					err = c.notifier.SendDeal(f)
				}

				if err != nil {
					log.Printf("Warning: failed to send deal: %v", err)
					continue
				}

				// Mark as sent
				c.detector.MarkSent(f)
				dealsFound++
			}
		}
	}

	// Update metadata
	c.store.SetMetadata("last_check", time.Now().Format(time.RFC3339))
	c.store.SetMetadata("deals_found", fmt.Sprintf("%d", dealsFound))

	log.Printf("Check complete. Found %d deals.", dealsFound)
	return nil
}

func (c *Checker) generateDateRanges(months []int) []dateRange {
	var ranges []dateRange
	now := time.Now()
	currentYear := now.Year()

	for _, month := range months {
		year := currentYear
		// If month is in the past this year, use next year
		if month < int(now.Month()) {
			year++
		}

		from := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
		to := from.AddDate(0, 1, -1) // Last day of month

		ranges = append(ranges, dateRange{from: from, to: to})
	}

	return ranges
}

type dateRange struct {
	from, to time.Time
}

func (c *Checker) deduplicateFlights(flights []api.Flight) []api.Flight {
	seen := make(map[string]api.Flight)

	for _, f := range flights {
		key := fmt.Sprintf("%s|%s|%s|%s",
			f.Origin, f.Destination,
			f.DepartureDate.Format("2006-01-02"),
			f.ReturnDate.Format("2006-01-02"))

		existing, ok := seen[key]
		if !ok || f.Price < existing.Price {
			seen[key] = f
		}
	}

	result := make([]api.Flight, 0, len(seen))
	for _, f := range seen {
		result = append(result, f)
	}
	return result
}

// storeAdapter adapts storage.Store to deals.Store interface
type storeAdapter struct {
	store *storage.Store
}

func (a *storeAdapter) GetAveragePrice(origin, dest, dep, ret string, days int) (float64, error) {
	return a.store.GetAveragePrice(origin, dest, dep, ret, days)
}

func (a *storeAdapter) IsDealSent(hash string) (bool, error) {
	return a.store.IsDealSent(hash)
}

func (a *storeAdapter) MarkDealSent(hash string) error {
	return a.store.MarkDealSent(hash)
}

func (a *storeAdapter) SavePrice(origin, dest, dep, ret string, price float64, currency, source string) error {
	return a.store.SavePrice(storage.PriceEntry{
		Origin:        origin,
		Destination:   dest,
		DepartureDate: dep,
		ReturnDate:    ret,
		Price:         price,
		Currency:      currency,
		APISource:     source,
	})
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/checker/...`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add internal/checker/checker.go
git commit -m "feat: implement checker orchestration"
```

---

### Task 8.2: Implement CLI Commands

**Files:**
- Create: `cmd/volato/main.go`

- [ ] **Step 1: Implement main.go with Cobra**

```go
// cmd/volato/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/override/volato/internal/checker"
	"github.com/override/volato/internal/config"
	"github.com/override/volato/internal/storage"
	"github.com/override/volato/internal/telegram"
)

var (
	cfgFile string
	dbFile  string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "volato",
		Short: "Flight deals monitor for Argentina",
	}

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ~/.config/volato/config.toml)")
	rootCmd.PersistentFlags().StringVar(&dbFile, "db", "", "database file (default: ~/.local/share/volato/volato.db)")

	rootCmd.AddCommand(checkCmd())
	rootCmd.AddCommand(botCmd())
	rootCmd.AddCommand(migrateCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func checkCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "Run flight check and send deal notifications",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, store, notifier, err := setup()
			if err != nil {
				return err
			}
			defer store.Close()

			c := checker.New(cfg, store, notifier)
			return c.Run(context.Background())
		},
	}
}

func botCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "bot",
		Short: "Start Telegram bot listener",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, store, notifier, err := setup()
			if err != nil {
				return err
			}
			defer store.Close()

			chatID, err := strconv.ParseInt(cfg.Telegram.ChatID, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid chat_id: %w", err)
			}

			bot, err := telegram.NewBot(cfg.Telegram.BotToken, chatID)
			if err != nil {
				return err
			}

			// Set up check function
			c := checker.New(cfg, store, notifier)
			bot.SetCheckFunc(func(ctx context.Context) error {
				return c.Run(ctx)
			})

			// Set up status function
			bot.SetStatusFunc(func() telegram.StatusInfo {
				lastCheck, _ := store.GetMetadata("last_check")
				dealsStr, _ := store.GetMetadata("deals_found")
				deals, _ := strconv.Atoi(dealsStr)

				var lastCheckTime time.Time
				if lastCheck != "" {
					lastCheckTime, _ = time.Parse(time.RFC3339, lastCheck)
				}

				return telegram.StatusInfo{
					LastCheck:  lastCheckTime,
					DealsFound: deals,
					NextCheck:  "Daily at 8:00 AM",
				}
			})

			// Set up deals function
			bot.SetDealsFunc(func() []string {
				entries, _ := store.GetRecentPrices(10)
				var deals []string
				for _, e := range entries {
					deals = append(deals, fmt.Sprintf("%s→%s: $%.0f (%s)",
						e.Origin, e.Destination, e.Price, e.DepartureDate))
				}
				return deals
			})

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Handle shutdown
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-sigCh
				log.Println("Shutting down...")
				cancel()
			}()

			log.Println("Bot started. Listening for commands...")
			return bot.Run(ctx)
		},
	}
}

func migrateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "migrate",
		Short: "Initialize or migrate the database",
		RunE: func(cmd *cobra.Command, args []string) error {
			dbPath := getDBPath()
			store, err := storage.New(dbPath)
			if err != nil {
				return err
			}
			store.Close()
			log.Printf("Database initialized at %s", dbPath)
			return nil
		},
	}
}

func setup() (*config.Config, *storage.Store, *telegram.Notifier, error) {
	cfgPath := getConfigPath()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("loading config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, nil, nil, fmt.Errorf("invalid config: %w", err)
	}

	dbPath := getDBPath()
	store, err := storage.New(dbPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("opening database: %w", err)
	}

	chatID, err := strconv.ParseInt(cfg.Telegram.ChatID, 10, 64)
	if err != nil {
		store.Close()
		return nil, nil, nil, fmt.Errorf("invalid chat_id: %w", err)
	}

	notifier, err := telegram.NewNotifier(cfg.Telegram.BotToken, chatID)
	if err != nil {
		store.Close()
		return nil, nil, nil, fmt.Errorf("creating notifier: %w", err)
	}

	return cfg, store, notifier, nil
}

func getConfigPath() string {
	if cfgFile != "" {
		return cfgFile
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "volato", "config.toml")
}

func getDBPath() string {
	if dbFile != "" {
		return dbFile
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "volato", "volato.db")
}
```

- [ ] **Step 2: Build and verify**

Run: `go build -o volato ./cmd/volato`
Expected: Binary created

- [ ] **Step 3: Test help output**

Run: `./volato --help`
Expected: Shows usage information

- [ ] **Step 4: Commit**

```bash
git add cmd/volato/main.go
git commit -m "feat: implement CLI with check, bot, and migrate commands"
```

---

## Chunk 9: Documentation & Example Config

### Task 9.1: Create Example Config

**Files:**
- Create: `config.example.toml`

- [ ] **Step 1: Create example config**

```toml
# Volato Configuration Example
# Copy this file to ~/.config/volato/config.toml and fill in your values

# Telegram Bot Settings
# Create a bot via @BotFather and get your chat ID from @userinfobot
[telegram]
bot_token = "YOUR_BOT_TOKEN"
chat_id = "YOUR_CHAT_ID"

# API Credentials
# At least one API must be configured

# Kiwi/Tequila API - https://tequila.kiwi.com/
[apis.kiwi]
api_key = "YOUR_KIWI_API_KEY"

# Amadeus API - https://developers.amadeus.com/
[apis.amadeus]
client_id = "YOUR_AMADEUS_CLIENT_ID"
client_secret = "YOUR_AMADEUS_CLIENT_SECRET"

# Default Settings
[defaults]
origin = "EZE"      # Buenos Aires Ezeiza (IATA code)
currency = "USD"    # Currency for prices

# Alert Settings
[alerts]
drop_threshold_percent = 20  # Alert when price drops 20% from average

# Flight Searches
# Add one [[searches]] block for each destination you want to monitor

[[searches]]
destination = "BCN"         # Barcelona
months = [6, 7, 8]          # June, July, August
max_price = 900             # Maximum price in USD

[searches.stay_days]
min = 10
max = 14

[[searches]]
destination = "FCO"         # Rome
origin = "AEP"              # Override: use Aeroparque instead of Ezeiza
months = [3, 4, 5]          # March, April, May
max_price = 850

[searches.stay_days]
min = 7
max = 10

[[searches]]
destination = "MIA"         # Miami
months = [12, 1, 2]         # December, January, February
max_price = 600

[searches.stay_days]
min = 5
max = 7
```

- [ ] **Step 2: Commit**

```bash
git add config.example.toml
git commit -m "docs: add example configuration file"
```

---

### Task 9.2: Create Setup Documentation

**Files:**
- Create: `docs/setup.md`

- [ ] **Step 1: Create setup.md**

```markdown
# Volato Setup Guide

This guide covers installing Volato on a Raspberry Pi 5.

## Prerequisites

- Raspberry Pi 5 with Raspberry Pi OS (64-bit)
- Internet connection
- API keys for Kiwi and/or Amadeus (see [apis.md](apis.md))
- Telegram bot token (see [apis.md](apis.md))

## Installation

### Option A: Build on Raspberry Pi

```bash
# Install Go (if not present)
sudo apt update
sudo apt install -y golang

# Clone the repository
git clone https://github.com/yourusername/volato.git
cd volato

# Build
go build -o volato ./cmd/volato

# Install
sudo mv volato /usr/local/bin/
```

### Option B: Cross-compile from Another Machine

```bash
# On your development machine
GOOS=linux GOARCH=arm64 go build -o volato ./cmd/volato

# Copy to Raspberry Pi
scp volato pi@raspberrypi:/home/pi/
ssh pi@raspberrypi "sudo mv /home/pi/volato /usr/local/bin/"
```

## Configuration

```bash
# Create config directory
mkdir -p ~/.config/volato
mkdir -p ~/.local/share/volato

# Copy and edit config
cp config.example.toml ~/.config/volato/config.toml
nano ~/.config/volato/config.toml
```

Edit the config file with your:
- Telegram bot token and chat ID
- Kiwi and/or Amadeus API keys
- Flight searches you want to monitor

See [configuration.md](configuration.md) for all options.

## Initialize Database

```bash
volato migrate
```

## Test the Setup

```bash
# Run a manual check
volato check
```

## Set Up Automatic Checks (Cron)

```bash
# Edit crontab
crontab -e

# Add this line to check daily at 8 AM:
0 8 * * * /usr/local/bin/volato check >> /var/log/volato.log 2>&1
```

## Set Up Telegram Bot (Systemd)

Create `/etc/systemd/system/volato-bot.service`:

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

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable volato-bot
sudo systemctl start volato-bot

# Check status
sudo systemctl status volato-bot

# View logs
journalctl -u volato-bot -f
```

## Verify Everything Works

1. Send `/status` to your Telegram bot
2. Send `/check` to trigger a manual check
3. Wait for the daily cron job to run

## Troubleshooting

### Bot not responding

```bash
# Check if bot is running
sudo systemctl status volato-bot

# Check logs
journalctl -u volato-bot --since "1 hour ago"
```

### Check not finding flights

```bash
# Run check with verbose output
volato check 2>&1 | tee /tmp/volato-debug.log
```

### Database issues

```bash
# Reinitialize database
rm ~/.local/share/volato/volato.db
volato migrate
```
```

- [ ] **Step 2: Commit**

```bash
git add docs/setup.md
git commit -m "docs: add Raspberry Pi setup guide"
```

---

### Task 9.3: Create APIs Documentation

**Files:**
- Create: `docs/apis.md`

- [ ] **Step 1: Create apis.md**

```markdown
# API Registration Guide

Volato requires API credentials for flight data and Telegram notifications.

## Kiwi (Tequila) API

Kiwi's Tequila API provides flight search with booking links.

**Free Tier:** 3,000 requests/month

### Registration Steps

1. Go to [tequila.kiwi.com](https://tequila.kiwi.com/)
2. Click "Get started for free"
3. Create an account
4. Go to "My Solutions" → "Create Solution"
5. Name your solution (e.g., "Volato")
6. Copy your API key from the dashboard

### Add to Config

```toml
[apis.kiwi]
api_key = "your_api_key_here"
```

---

## Amadeus API

Amadeus provides comprehensive flight data with broader airline coverage.

**Free Tier:** 2,000 calls/month

### Registration Steps

1. Go to [developers.amadeus.com](https://developers.amadeus.com/)
2. Click "Get Started"
3. Create an account
4. Go to "My Apps" → "Create App"
5. Name your app (e.g., "Volato")
6. Select "Production" environment for real data
7. Copy your Client ID and Client Secret

### Add to Config

```toml
[apis.amadeus]
client_id = "your_client_id"
client_secret = "your_client_secret"
```

---

## Telegram Bot

The Telegram bot sends deal notifications and accepts commands.

### Create Bot

1. Open Telegram and search for `@BotFather`
2. Send `/newbot`
3. Follow the prompts to name your bot
4. Copy the bot token (looks like `123456:ABC-DEF...`)

### Get Your Chat ID

1. Search for `@userinfobot` on Telegram
2. Start a conversation
3. It will reply with your user ID (this is your chat_id)

Alternatively, after setting up your bot:
1. Start a conversation with your bot
2. Visit `https://api.telegram.org/bot<YOUR_BOT_TOKEN>/getUpdates`
3. Find your chat_id in the response

### Add to Config

```toml
[telegram]
bot_token = "123456:ABC-DEF..."
chat_id = "your_chat_id"
```

### Test Your Bot

After starting `volato bot`, send `/help` to your bot. It should respond with available commands.

---

## API Usage Tips

- **Rate Limits:** With daily checks, you'll use ~30 API calls per day per destination
- **Kiwi Primary:** Kiwi is used first as it provides booking links
- **Amadeus Backup:** Amadeus is queried second for additional coverage
- **Quota Check:** Use `/status` in Telegram to monitor usage
```

- [ ] **Step 2: Commit**

```bash
git add docs/apis.md
git commit -m "docs: add API registration guide"
```

---

### Task 9.4: Create Configuration Reference

**Files:**
- Create: `docs/configuration.md`

- [ ] **Step 1: Create configuration.md**

```markdown
# Configuration Reference

Volato uses a TOML configuration file located at:
- `~/.config/volato/config.toml` (default)
- Custom path via `--config` flag

## Full Configuration Example

```toml
[telegram]
bot_token = "123456:ABC-DEF..."
chat_id = "999888777"

[apis.kiwi]
api_key = "your_kiwi_key"

[apis.amadeus]
client_id = "your_client_id"
client_secret = "your_client_secret"

[defaults]
origin = "EZE"
currency = "USD"

[alerts]
drop_threshold_percent = 20

[[searches]]
destination = "BCN"
months = [6, 7, 8]
max_price = 900
[searches.stay_days]
min = 10
max = 14
```

## Section Reference

### [telegram]

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `bot_token` | string | Yes | Telegram bot token from @BotFather |
| `chat_id` | string | Yes | Your Telegram user ID |

### [apis.kiwi]

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `api_key` | string | No* | Kiwi/Tequila API key |

### [apis.amadeus]

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `client_id` | string | No* | Amadeus client ID |
| `client_secret` | string | No* | Amadeus client secret |

*At least one API (Kiwi or Amadeus) must be configured.

### [defaults]

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `origin` | string | Yes | - | Default departure airport (IATA code) |
| `currency` | string | Yes | - | Currency for prices (e.g., "USD", "EUR") |

### [alerts]

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `drop_threshold_percent` | int | No | 20 | Alert when price drops this % from average |

### [[searches]]

Each search defines a destination to monitor. Add multiple `[[searches]]` blocks.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `destination` | string | Yes | Destination airport (IATA code) |
| `origin` | string | No | Override default origin for this search |
| `months` | int[] | Yes | Months to search (1-12) |
| `max_price` | float | Yes | Maximum price threshold for alerts |
| `stay_days.min` | int | Yes | Minimum trip duration in days |
| `stay_days.max` | int | Yes | Maximum trip duration in days |

## Common Airport Codes (Argentina)

| Code | Airport |
|------|---------|
| EZE | Buenos Aires Ezeiza International |
| AEP | Buenos Aires Aeroparque |
| COR | Córdoba |
| MDZ | Mendoza |
| ROS | Rosario |

## Common Destination Codes

| Code | City |
|------|------|
| BCN | Barcelona |
| MAD | Madrid |
| FCO | Rome |
| CDG | Paris |
| MIA | Miami |
| JFK | New York |
| LAX | Los Angeles |
| LHR | London |

## CLI Flags

```bash
volato [command] [flags]

Flags:
  --config string   Config file path (default ~/.config/volato/config.toml)
  --db string       Database file path (default ~/.local/share/volato/volato.db)

Commands:
  check     Run flight check and send notifications
  bot       Start Telegram bot listener
  migrate   Initialize or migrate the database
```
```

- [ ] **Step 2: Commit**

```bash
git add docs/configuration.md
git commit -m "docs: add configuration reference"
```

---

### Task 9.5: Final Cleanup and Tidy

**Files:**
- All

- [ ] **Step 1: Run go mod tidy**

```bash
go mod tidy
```

- [ ] **Step 2: Run go vet**

```bash
go vet ./...
```

- [ ] **Step 3: Run all tests**

```bash
go test ./... -v
```

- [ ] **Step 4: Build final binary**

```bash
go build -o volato ./cmd/volato
```

- [ ] **Step 5: Final commit**

```bash
git add -A
git commit -m "chore: final cleanup and verification"
```

---

## Summary

This plan creates a complete flight deals bot with:

- **9 chunks** of work
- **~25 tasks** following TDD (red-green-refactor)
- Full test coverage for core components
- CLI with check, bot, and migrate commands
- SQLite storage for price history
- Kiwi and Amadeus API integration
- Telegram notifications and commands
- Raspberry Pi deployment documentation

**Estimated API calls per check:** ~2-4 per destination (Kiwi + Amadeus per date range)

**With daily checks and 5 destinations:** ~600-1200 calls/month (well within free tiers)
