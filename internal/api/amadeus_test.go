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
