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
					deals = append(deals, fmt.Sprintf("%s->%s: $%.0f (%s)",
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
