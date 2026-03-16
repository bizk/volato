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
