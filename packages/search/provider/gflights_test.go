package provider

import (
	"context"
	"testing"
	"time"

	"golang.org/x/text/currency"
)

func TestGFlightsSearch(t *testing.T) {
	provider, err := NewGFlights()
	if err != nil {
		t.Fatal(err)
	}

	itineries, err := provider.Search(
		context.Background(),
		Request{
			Origin:        "LONDON",
			Destination:   "NEW YORK",
			DepartureDate: time.Now().Add(time.Hour * 24),
			ReturnDate:    time.Now().Add(time.Hour * 24 * 7),
			Adults:        1,
			Class:         Economy,
			Currency:      currency.GBP,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if len(itineries) == 0 {
		t.Fatal("expected itineries, got none")
	}
}
