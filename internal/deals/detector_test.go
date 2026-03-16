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
