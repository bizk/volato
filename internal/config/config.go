package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// TelegramConfig holds Telegram bot configuration
type TelegramConfig struct {
	BotToken string `toml:"bot_token"`
	ChatID   string `toml:"chat_id"`
}

// KiwiConfig holds Kiwi API configuration
type KiwiConfig struct {
	APIKey string `toml:"api_key"`
}

// AmadeusConfig holds Amadeus API configuration
type AmadeusConfig struct {
	ClientID     string `toml:"client_id"`
	ClientSecret string `toml:"client_secret"`
}

// APIsConfig holds all API configurations
type APIsConfig struct {
	Kiwi     KiwiConfig     `toml:"kiwi"`
	Amadeus  AmadeusConfig  `toml:"amadeus"`
}

// DefaultsConfig holds default values for searches
type DefaultsConfig struct {
	Origin   string `toml:"origin"`
	Currency string `toml:"currency"`
}

// AlertsConfig holds alert configuration
type AlertsConfig struct {
	DropThresholdPercent float64 `toml:"drop_threshold_percent"`
}

// StayDays holds min and max stay duration
type StayDays struct {
	Min int `toml:"min"`
	Max int `toml:"max"`
}

// SearchConfig holds a single search configuration
type SearchConfig struct {
	Destination string   `toml:"destination"`
	Origin      string   `toml:"origin"`
	Months      []string `toml:"months"`
	StayDays    StayDays `toml:"stay_days"`
	MaxPrice    float64  `toml:"max_price"`
}

// Config holds the entire application configuration
type Config struct {
	Telegram TelegramConfig `toml:"telegram"`
	APIs     APIsConfig     `toml:"apis"`
	Defaults DefaultsConfig `toml:"defaults"`
	Alerts   AlertsConfig   `toml:"alerts"`
	Searches []SearchConfig `toml:"searches"`
}

// Load reads and parses a TOML configuration file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	err = toml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// Validate checks that the configuration is valid
func (c *Config) Validate() error {
	// Check Telegram config
	if c.Telegram.BotToken == "" {
		return fmt.Errorf("telegram.bot_token is required")
	}
	if c.Telegram.ChatID == "" {
		return fmt.Errorf("telegram.chat_id is required")
	}

	// Check at least one API is configured
	if c.APIs.Kiwi.APIKey == "" && c.APIs.Amadeus.ClientID == "" {
		return fmt.Errorf("at least one API must be configured (kiwi or amadeus)")
	}

	// Check Defaults config
	if c.Defaults.Origin == "" {
		return fmt.Errorf("defaults.origin is required")
	}
	if c.Defaults.Currency == "" {
		return fmt.Errorf("defaults.currency is required")
	}

	// Check at least one search exists
	if len(c.Searches) == 0 {
		return fmt.Errorf("at least one search is required")
	}

	// Validate each search
	for i, search := range c.Searches {
		if search.Destination == "" {
			return fmt.Errorf("search[%d].destination is required", i)
		}
		if len(search.Months) == 0 {
			return fmt.Errorf("search[%d].months is required", i)
		}
		if search.StayDays.Min <= 0 || search.StayDays.Max <= 0 {
			return fmt.Errorf("search[%d].stay_days.min and max must be positive", i)
		}
		if search.StayDays.Min > search.StayDays.Max {
			return fmt.Errorf("search[%d].stay_days.min must be <= max", i)
		}
		if search.MaxPrice <= 0 {
			return fmt.Errorf("search[%d].max_price must be positive", i)
		}
	}

	return nil
}

// EffectiveOrigin returns the search's origin if set, otherwise the default origin
func (c *Config) EffectiveOrigin(s *SearchConfig) string {
	if s.Origin != "" {
		return s.Origin
	}
	return c.Defaults.Origin
}
