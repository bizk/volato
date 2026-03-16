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
	store            Store
	dropThresholdPct int
	historyDays      int
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

	// Check for price drop first (more specific reason)
	avgPrice, err := d.store.GetAveragePrice(f.Origin, f.Destination, depDate, retDate, d.historyDays)
	if err == nil && avgPrice > 0 {
		dropPct := ((avgPrice - f.Price) / avgPrice) * 100
		if dropPct >= float64(d.dropThresholdPct) {
			return EvaluationResult{
				IsDeal:       true,
				Reason:       ReasonPriceDrop,
				DropPercent:  dropPct,
				AveragePrice: avgPrice,
			}
		}
	}

	// Check price threshold
	if f.Price <= maxPrice {
		return EvaluationResult{
			IsDeal: true,
			Reason: ReasonBelowThreshold,
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
