package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_ValidConfig(t *testing.T) {
	// Create a temporary TOML config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	configContent := `
[telegram]
bot_token = "test-token-123"
chat_id = "123456789"

[apis.kiwi]
api_key = "kiwi-key"

[apis.amadeus]
client_id = "amadeus-id"
client_secret = "amadeus-secret"

[defaults]
origin = "NYC"
currency = "USD"

[alerts]
drop_threshold_percent = 20.0

[[searches]]
destination = "LHR"
origin = "NYC"
months = ["03", "04", "05"]
stay_days = { min = 5, max = 10 }
max_price = 1000.0

[[searches]]
destination = "CDG"
months = ["06", "07"]
stay_days = { min = 7, max = 14 }
max_price = 1200.0
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	config, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load valid config: %v", err)
	}

	// Verify Telegram config
	if config.Telegram.BotToken != "test-token-123" {
		t.Errorf("Expected bot_token 'test-token-123', got '%s'", config.Telegram.BotToken)
	}
	if config.Telegram.ChatID != "123456789" {
		t.Errorf("Expected chat_id '123456789', got '%s'", config.Telegram.ChatID)
	}

	// Verify APIs config
	if config.APIs.Kiwi.APIKey != "kiwi-key" {
		t.Errorf("Expected Kiwi API key 'kiwi-key', got '%s'", config.APIs.Kiwi.APIKey)
	}
	if config.APIs.Amadeus.ClientID != "amadeus-id" {
		t.Errorf("Expected Amadeus ClientID 'amadeus-id', got '%s'", config.APIs.Amadeus.ClientID)
	}
	if config.APIs.Amadeus.ClientSecret != "amadeus-secret" {
		t.Errorf("Expected Amadeus ClientSecret 'amadeus-secret', got '%s'", config.APIs.Amadeus.ClientSecret)
	}

	// Verify Defaults config
	if config.Defaults.Origin != "NYC" {
		t.Errorf("Expected defaults origin 'NYC', got '%s'", config.Defaults.Origin)
	}
	if config.Defaults.Currency != "USD" {
		t.Errorf("Expected defaults currency 'USD', got '%s'", config.Defaults.Currency)
	}

	// Verify Alerts config
	if config.Alerts.DropThresholdPercent != 20.0 {
		t.Errorf("Expected drop_threshold_percent 20.0, got %f", config.Alerts.DropThresholdPercent)
	}

	// Verify Searches config
	if len(config.Searches) != 2 {
		t.Fatalf("Expected 2 searches, got %d", len(config.Searches))
	}

	// Check first search
	if config.Searches[0].Destination != "LHR" {
		t.Errorf("Expected first search destination 'LHR', got '%s'", config.Searches[0].Destination)
	}
	if config.Searches[0].Origin != "NYC" {
		t.Errorf("Expected first search origin 'NYC', got '%s'", config.Searches[0].Origin)
	}
	if len(config.Searches[0].Months) != 3 {
		t.Errorf("Expected first search to have 3 months, got %d", len(config.Searches[0].Months))
	}
	if config.Searches[0].StayDays.Min != 5 || config.Searches[0].StayDays.Max != 10 {
		t.Errorf("Expected first search stay_days min=5 max=10, got min=%d max=%d", config.Searches[0].StayDays.Min, config.Searches[0].StayDays.Max)
	}
	if config.Searches[0].MaxPrice != 1000.0 {
		t.Errorf("Expected first search max_price 1000.0, got %f", config.Searches[0].MaxPrice)
	}

	// Check second search
	if config.Searches[1].Destination != "CDG" {
		t.Errorf("Expected second search destination 'CDG', got '%s'", config.Searches[1].Destination)
	}
	if config.Searches[1].StayDays.Min != 7 || config.Searches[1].StayDays.Max != 14 {
		t.Errorf("Expected second search stay_days min=7 max=14, got min=%d max=%d", config.Searches[1].StayDays.Min, config.Searches[1].StayDays.Max)
	}
}

func TestLoadConfig_MissingFile(t *testing.T) {
	_, err := Load("/nonexistent/path/config.toml")
	if err == nil {
		t.Fatal("Expected error when loading non-existent file, got nil")
	}
}

func TestLoadConfig_InvalidTOML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.toml")

	invalidContent := `
[telegram
bot_token = "test"
`

	err := os.WriteFile(configPath, []byte(invalidContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	_, err = Load(configPath)
	if err == nil {
		t.Fatal("Expected error when loading invalid TOML, got nil")
	}
}

func TestConfig_Validate_MissingBotToken(t *testing.T) {
	config := &Config{
		Telegram: TelegramConfig{
			BotToken: "",
			ChatID:   "123456789",
		},
		APIs: APIsConfig{
			Kiwi: KiwiConfig{APIKey: "key"},
		},
		Defaults: DefaultsConfig{
			Origin:   "NYC",
			Currency: "USD",
		},
		Searches: []SearchConfig{
			{
				Destination: "LHR",
				Months:      []string{"03"},
				StayDays:    StayDays{Min: 5, Max: 10},
				MaxPrice:    1000.0,
			},
		},
	}

	err := config.Validate()
	if err == nil {
		t.Fatal("Expected validation error for missing bot_token, got nil")
	}
}

func TestConfig_Validate_NoSearches(t *testing.T) {
	config := &Config{
		Telegram: TelegramConfig{
			BotToken: "token",
			ChatID:   "123456789",
		},
		APIs: APIsConfig{
			Kiwi: KiwiConfig{APIKey: "key"},
		},
		Defaults: DefaultsConfig{
			Origin:   "NYC",
			Currency: "USD",
		},
		Searches: []SearchConfig{},
	}

	err := config.Validate()
	if err == nil {
		t.Fatal("Expected validation error for no searches, got nil")
	}
}

func TestConfig_Validate_Valid(t *testing.T) {
	config := &Config{
		Telegram: TelegramConfig{
			BotToken: "token",
			ChatID:   "123456789",
		},
		APIs: APIsConfig{
			Kiwi: KiwiConfig{APIKey: "key"},
		},
		Defaults: DefaultsConfig{
			Origin:   "NYC",
			Currency: "USD",
		},
		Searches: []SearchConfig{
			{
				Destination: "LHR",
				Months:      []string{"03"},
				StayDays:    StayDays{Min: 5, Max: 10},
				MaxPrice:    1000.0,
			},
		},
	}

	err := config.Validate()
	if err != nil {
		t.Fatalf("Expected no validation error for valid config, got: %v", err)
	}
}

func TestConfig_EffectiveOrigin(t *testing.T) {
	config := &Config{
		Defaults: DefaultsConfig{
			Origin: "NYC",
		},
	}

	search := SearchConfig{
		Origin: "LAX",
	}

	origin := config.EffectiveOrigin(&search)
	if origin != "LAX" {
		t.Errorf("Expected origin 'LAX' from search, got '%s'", origin)
	}

	search2 := SearchConfig{
		Origin: "",
	}

	origin2 := config.EffectiveOrigin(&search2)
	if origin2 != "NYC" {
		t.Errorf("Expected origin 'NYC' from defaults, got '%s'", origin2)
	}
}
