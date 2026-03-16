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
