package provider_test

import (
	"context"
	"testing"
	"time"

	"github.com/tobyrushton/flyvia/packages/env"
	"github.com/tobyrushton/flyvia/packages/search/provider"
	"golang.org/x/text/currency"
)

func TestGFlightsSearch(t *testing.T) {
	cfg, err := env.Load("../../../.env")
	if err != nil {
		t.Skipf("skipping: could not load env: %v", err)
	}
	p, err := provider.NewGFlights(cfg.ProxyURL)
	if err != nil {
		t.Fatal(err)
	}

	itineries, err := p.Search(
		context.Background(),
		provider.Request{
			Origin:        "LONDON",
			Destination:   "NEW YORK",
			DepartureDate: time.Now().Add(time.Hour * 24),
			ReturnDate:    time.Now().Add(time.Hour * 24 * 7),
			Adults:        1,
			Class:         provider.Economy,
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

func TestGFlightsExplore(t *testing.T) {
	cfg, err := env.Load("../../../.env")
	if err != nil {
		t.Skipf("skipping: could not load env: %v", err)
	}
	p, err := provider.NewGFlights(cfg.ProxyURL)
	if err != nil {
		t.Fatal(err)
	}

	itineries, err := p.Explore(
		context.Background(),
		provider.Request{
			Origin:        "LONDON",
			DepartureDate: time.Now().Add(time.Hour * 24),
			ReturnDate:    time.Now().Add(time.Hour * 24 * 7),
			Adults:        1,
			Class:         provider.Economy,
			Currency:      currency.GBP,
		},
		"London",
	)
	if err != nil {
		t.Fatal(err)
	}

	if len(itineries) == 0 {
		t.Fatal("expected itineries, got none")
	}
}
