// internal/api/kiwi.go
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const defaultKiwiBaseURL = "https://api.tequila.kiwi.com"

// KiwiClient implements FlightSearcher for Kiwi/Tequila API.
type KiwiClient struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// NewKiwiClient creates a new Kiwi API client.
func NewKiwiClient(apiKey, baseURL string) *KiwiClient {
	if baseURL == "" {
		baseURL = defaultKiwiBaseURL
	}
	return &KiwiClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

// Name returns the API name.
func (c *KiwiClient) Name() string {
	return "kiwi"
}

// kiwiResponse represents the Kiwi API response structure.
type kiwiResponse struct {
	Data     []kiwiFlightData `json:"data"`
	Currency string           `json:"currency"`
}

type kiwiFlightData struct {
	FlyFrom        string   `json:"flyFrom"`
	FlyTo          string   `json:"flyTo"`
	Price          float64  `json:"price"`
	DeepLink       string   `json:"deep_link"`
	LocalDeparture string   `json:"local_departure"`
	LocalArrival   string   `json:"local_arrival"`
	Airlines       []string `json:"airlines"`
	Route          []struct {
		Airline string `json:"airline"`
	} `json:"route"`
}

// Search queries the Kiwi API for flights.
func (c *KiwiClient) Search(ctx context.Context, req SearchRequest) ([]Flight, error) {
	u, err := url.Parse(c.baseURL + "/v2/search")
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("fly_from", req.Origin)
	q.Set("fly_to", req.Destination)
	q.Set("date_from", req.DateFrom.Format("02/01/2006"))
	q.Set("date_to", req.DateTo.Format("02/01/2006"))
	q.Set("nights_in_dst_from", fmt.Sprintf("%d", req.StayDaysMin))
	q.Set("nights_in_dst_to", fmt.Sprintf("%d", req.StayDaysMax))
	q.Set("curr", req.Currency)
	q.Set("flight_type", "round")
	q.Set("one_for_city", "0")
	q.Set("partner", "picky")
	u.RawQuery = q.Encode()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("apikey", c.apiKey)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("kiwi request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("kiwi returned status %d", resp.StatusCode)
	}

	var kiwiResp kiwiResponse
	if err := json.NewDecoder(resp.Body).Decode(&kiwiResp); err != nil {
		return nil, fmt.Errorf("decoding kiwi response: %w", err)
	}

	flights := make([]Flight, 0, len(kiwiResp.Data))
	for _, d := range kiwiResp.Data {
		depTime, _ := time.Parse(time.RFC3339, d.LocalDeparture)

		// Calculate return date from route (simplified - last segment)
		var retTime time.Time
		if len(d.Route) > 0 {
			retTime = depTime.AddDate(0, 0, req.StayDaysMin) // Approximation
		}

		airline := ""
		if len(d.Airlines) > 0 {
			airline = d.Airlines[0]
		}

		flights = append(flights, Flight{
			Origin:        d.FlyFrom,
			Destination:   d.FlyTo,
			DepartureDate: depTime,
			ReturnDate:    retTime,
			Price:         d.Price,
			Currency:      kiwiResp.Currency,
			Airline:       airline,
			Stops:         len(d.Route) - 1,
			BookingLink:   d.DeepLink,
			APISource:     "kiwi",
		})
	}

	return flights, nil
}
