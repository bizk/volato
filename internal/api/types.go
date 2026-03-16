package api

import (
	"context"
	"time"
)

type Flight struct {
	Origin        string
	Destination   string
	DepartureDate time.Time
	ReturnDate    time.Time
	Price         float64
	Currency      string
	Airline       string
	Stops         int
	BookingLink   string
	APISource     string
}

type SearchRequest struct {
	Origin      string
	Destination string
	DateFrom    time.Time
	DateTo      time.Time
	StayDaysMin int
	StayDaysMax int
	Currency    string
}

type FlightSearcher interface {
	Search(ctx context.Context, req SearchRequest) ([]Flight, error)
	Name() string
}
