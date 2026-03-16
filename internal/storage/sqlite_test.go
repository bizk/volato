package storage

import (
	"os"
	"testing"
	"time"
)

func newTestDB(t *testing.T) *Store {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "test_*.db")
	if err != nil {
		t.Fatalf("failed to create temp db file: %v", err)
	}
	tmpFile.Close()
	dbPath := tmpFile.Name()

	store, err := New(dbPath)
	if err != nil {
		os.Remove(dbPath)
		t.Fatalf("failed to create store: %v", err)
	}

	t.Cleanup(func() {
		store.Close()
		os.Remove(dbPath)
	})

	return store
}

func TestStore_SaveAndGetPriceHistory(t *testing.T) {
	store := newTestDB(t)

	entry := PriceEntry{
		Origin:        "JFK",
		Destination:   "LHR",
		DepartureDate: "2024-03-20",
		ReturnDate:    "2024-03-27",
		Price:         450.0,
		Currency:      "USD",
		APISource:     "kiwi",
	}

	err := store.SavePrice(entry)
	if err != nil {
		t.Fatalf("failed to save price: %v", err)
	}

	// Save another price to test averaging
	entry2 := PriceEntry{
		Origin:        "JFK",
		Destination:   "LHR",
		DepartureDate: "2024-03-20",
		ReturnDate:    "2024-03-27",
		Price:         550.0,
		Currency:      "USD",
		APISource:     "amadeus",
	}
	err = store.SavePrice(entry2)
	if err != nil {
		t.Fatalf("failed to save second price: %v", err)
	}

	// Get average price for the past 7 days
	avg, err := store.GetAveragePrice("JFK", "LHR", "2024-03-20", "2024-03-27", 7)
	if err != nil {
		t.Fatalf("failed to get average price: %v", err)
	}

	expected := 500.0 // (450 + 550) / 2
	if avg != expected {
		t.Errorf("expected average %f, got %f", expected, avg)
	}
}

func TestStore_DealDeduplication(t *testing.T) {
	store := newTestDB(t)

	hash := "flight_hash_12345"

	// Should not be sent initially
	sent, err := store.IsDealSent(hash)
	if err != nil {
		t.Fatalf("failed to check if deal sent: %v", err)
	}
	if sent {
		t.Error("expected deal to not be sent initially")
	}

	// Mark as sent
	err = store.MarkDealSent(hash)
	if err != nil {
		t.Fatalf("failed to mark deal as sent: %v", err)
	}

	// Should now be sent
	sent, err = store.IsDealSent(hash)
	if err != nil {
		t.Fatalf("failed to check if deal sent after marking: %v", err)
	}
	if !sent {
		t.Error("expected deal to be sent after marking")
	}
}

func TestStore_GetRecentDeals(t *testing.T) {
	store := newTestDB(t)

	// Insert multiple price entries
	for i := 0; i < 5; i++ {
		entry := PriceEntry{
			Origin:        "JFK",
			Destination:   "LHR",
			DepartureDate: "2024-03-20",
			ReturnDate:    "2024-03-27",
			Price:         float64(400 + i*10),
			Currency:      "USD",
			APISource:     "kiwi",
		}
		err := store.SavePrice(entry)
		if err != nil {
			t.Fatalf("failed to save price: %v", err)
		}
	}

	// Get recent prices with limit
	prices, err := store.GetRecentPrices(3)
	if err != nil {
		t.Fatalf("failed to get recent prices: %v", err)
	}

	if len(prices) != 3 {
		t.Errorf("expected 3 prices, got %d", len(prices))
	}
}

func TestStore_Cleanup(t *testing.T) {
	store := newTestDB(t)

	// Insert some old data
	entry := PriceEntry{
		Origin:        "JFK",
		Destination:   "LHR",
		DepartureDate: "2024-03-20",
		ReturnDate:    "2024-03-27",
		Price:         450.0,
		Currency:      "USD",
		APISource:     "kiwi",
	}
	err := store.SavePrice(entry)
	if err != nil {
		t.Fatalf("failed to save price: %v", err)
	}

	hash := "test_hash"
	err = store.MarkDealSent(hash)
	if err != nil {
		t.Fatalf("failed to mark deal sent: %v", err)
	}

	// Cleanup should run without error
	err = store.Cleanup(7, 30)
	if err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}
}

func TestStore_Metadata(t *testing.T) {
	store := newTestDB(t)

	key := "last_check"
	value := time.Now().Format(time.RFC3339)

	// Set metadata
	err := store.SetMetadata(key, value)
	if err != nil {
		t.Fatalf("failed to set metadata: %v", err)
	}

	// Get metadata
	retrieved, err := store.GetMetadata(key)
	if err != nil {
		t.Fatalf("failed to get metadata: %v", err)
	}

	if retrieved != value {
		t.Errorf("expected metadata %s, got %s", value, retrieved)
	}

	// Get non-existent key should return empty string and error
	_, err = store.GetMetadata("non_existent")
	if err == nil {
		t.Error("expected error when getting non-existent metadata")
	}
}
