// internal/api/amadeus.go
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const defaultAmadeusBaseURL = "https://api.amadeus.com"

// AmadeusClient implements FlightSearcher for Amadeus API.
type AmadeusClient struct {
	clientID     string
	clientSecret string
	baseURL      string
	client       *http.Client

	mu          sync.Mutex
	accessToken string
	tokenExpiry time.Time
}

// NewAmadeusClient creates a new Amadeus API client.
func NewAmadeusClient(clientID, clientSecret, baseURL string) *AmadeusClient {
	if baseURL == "" {
		baseURL = defaultAmadeusBaseURL
	}
	return &AmadeusClient{
		clientID:     clientID,
		clientSecret: clientSecret,
		baseURL:      baseURL,
		client:       &http.Client{Timeout: 30 * time.Second},
	}
}

// Name returns the API name.
func (c *AmadeusClient) Name() string {
	return "amadeus"
}

type amadeusTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

type amadeusFlightResponse struct {
	Data []amadeusFlightOffer `json:"data"`
}

type amadeusFlightOffer struct {
	Price       amadeusPrice       `json:"price"`
	Itineraries []amadeusItinerary `json:"itineraries"`
}

type amadeusPrice struct {
	Total    string `json:"total"`
	Currency string `json:"currency"`
}

type amadeusItinerary struct {
	Segments []amadeusSegment `json:"segments"`
}

type amadeusSegment struct {
	Departure   amadeusLocation `json:"departure"`
	Arrival     amadeusLocation `json:"arrival"`
	CarrierCode string          `json:"carrierCode"`
}

type amadeusLocation struct {
	IataCode string `json:"iataCode"`
	At       string `json:"at"`
}

func (c *AmadeusClient) getToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Return cached token if still valid
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry) {
		return c.accessToken, nil
	}

	// Request new token
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", c.clientID)
	data.Set("client_secret", c.clientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/v1/security/oauth2/token",
		strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token request returned status %d", resp.StatusCode)
	}

	var tokenResp amadeusTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}

	c.accessToken = tokenResp.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn-60) * time.Second)

	return c.accessToken, nil
}

// Search queries the Amadeus API for flights.
func (c *AmadeusClient) Search(ctx context.Context, req SearchRequest) ([]Flight, error) {
	token, err := c.getToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting token: %w", err)
	}

	u, err := url.Parse(c.baseURL + "/v2/shopping/flight-offers")
	if err != nil {
		return nil, err
	}

	// Amadeus requires specific departure date, not range
	// We'll search for the start date
	departureDate := req.DateFrom.Format("2006-01-02")
	returnDate := req.DateFrom.AddDate(0, 0, req.StayDaysMin).Format("2006-01-02")

	q := u.Query()
	q.Set("originLocationCode", req.Origin)
	q.Set("destinationLocationCode", req.Destination)
	q.Set("departureDate", departureDate)
	q.Set("returnDate", returnDate)
	q.Set("adults", "1")
	q.Set("currencyCode", req.Currency)
	q.Set("max", "50")
	u.RawQuery = q.Encode()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("amadeus request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("amadeus returned status %d", resp.StatusCode)
	}

	var amadeusResp amadeusFlightResponse
	if err := json.NewDecoder(resp.Body).Decode(&amadeusResp); err != nil {
		return nil, fmt.Errorf("decoding amadeus response: %w", err)
	}

	flights := make([]Flight, 0, len(amadeusResp.Data))
	for _, offer := range amadeusResp.Data {
		if len(offer.Itineraries) < 2 {
			continue // Need outbound and return
		}

		outbound := offer.Itineraries[0]
		inbound := offer.Itineraries[1]

		if len(outbound.Segments) == 0 || len(inbound.Segments) == 0 {
			continue
		}

		firstSeg := outbound.Segments[0]
		lastReturnSeg := inbound.Segments[len(inbound.Segments)-1]

		depTime, _ := time.Parse("2006-01-02T15:04:05", firstSeg.Departure.At)
		retTime, _ := time.Parse("2006-01-02T15:04:05", lastReturnSeg.Arrival.At)

		price, _ := strconv.ParseFloat(offer.Price.Total, 64)

		// Count stops (segments - 1 for each direction)
		stops := (len(outbound.Segments) - 1) + (len(inbound.Segments) - 1)

		flights = append(flights, Flight{
			Origin:        firstSeg.Departure.IataCode,
			Destination:   firstSeg.Arrival.IataCode,
			DepartureDate: depTime,
			ReturnDate:    retTime,
			Price:         price,
			Currency:      offer.Price.Currency,
			Airline:       firstSeg.CarrierCode,
			Stops:         stops,
			BookingLink:   "", // Amadeus doesn't provide direct booking links in basic API
			APISource:     "amadeus",
		})
	}

	return flights, nil
}
