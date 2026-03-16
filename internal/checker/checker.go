// internal/checker/checker.go
package checker

import (
	"context"
	"fmt"
	"log"
	"strconv"
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
	detector := deals.New(storeAdapter, int(cfg.Alerts.DropThresholdPercent))

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

	for i := range c.cfg.Searches {
		search := &c.cfg.Searches[i]
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

func (c *Checker) generateDateRanges(months []string) []dateRange {
	var ranges []dateRange
	now := time.Now()
	currentYear := now.Year()

	for _, monthStr := range months {
		month, err := strconv.Atoi(monthStr)
		if err != nil {
			log.Printf("Warning: invalid month %q, skipping", monthStr)
			continue
		}

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
