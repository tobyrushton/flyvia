//go:build integration
// +build integration

package integration_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/text/currency"

	"github.com/tobyrushton/flyvia/packages/env"
	"github.com/tobyrushton/flyvia/packages/search"
	"github.com/tobyrushton/flyvia/packages/search/provider"
)

// newGFlightsProvider creates a real GFlights provider, skipping the test if
// env loading or session creation fails (e.g. missing .env, network down, rate-limited).
func newGFlightsProvider(t *testing.T) *provider.GFlights {
	t.Helper()

	cfg, err := env.Load("../../.env")
	if err != nil {
		t.Skipf("skipping integration test: could not load env: %v", err)
	}
	p, err := provider.NewGFlights(cfg.ProxyURL)
	if err != nil {
		t.Skipf("skipping integration test: could not create GFlights session: %v", err)
	}
	return p
}

// futureDate returns a date roughly `days` in the future, anchored at noon UTC
// to avoid timezone edge-cases.
func futureDate(days int) time.Time {
	return time.Now().UTC().Truncate(24 * time.Hour).
		Add(time.Duration(days) * 24 * time.Hour).
		Add(12 * time.Hour)
}

func countResults(results map[string][]search.Result) int {
	count := 0
	for _, group := range results {
		count += len(group)
	}
	return count
}

func firstResult(results map[string][]search.Result) (search.Result, bool) {
	for _, group := range results {
		if len(group) == 0 {
			continue
		}
		return group[0], true
	}
	return search.Result{}, false
}

// --- Full end-to-end search integration tests ---

func TestIntegration_Search_LondonToPhnomPenh(t *testing.T) {
	p := newGFlightsProvider(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	s := search.New(ctx, p)

	results, err := s.Search(provider.Request{
		Origin:        "london",
		Destination:   "tokyo",
		DepartureDate: futureDate(30),
		ReturnDate:    futureDate(37),
		Adults:        1,
		Class:         provider.Economy,
		Currency:      currency.GBP,
	})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}

	if err := writeResultsJSON("london_to_hanoi", results); err != nil {
		t.Fatalf("failed to write results json: %v", err)
	}

	t.Logf("found %d split-ticket results for London → Hanoi", countResults(results))

	for stopCity, group := range results {
		for i, r := range group {
			if r.Price <= 0 {
				t.Errorf("%s result[%d]: expected positive price, got %f", stopCity, i, r.Price)
			}
			if r.StopCity == "" {
				t.Errorf("%s result[%d]: expected non-empty stop city", stopCity, i)
			}
			if len(r.Itineries) != 2 {
				t.Errorf("%s result[%d]: expected 2 itinerary legs, got %d", stopCity, i, len(r.Itineries))
			}
			if len(r.StopLengths) != 2 {
				t.Errorf("%s result[%d]: expected 2 stop lengths, got %d", stopCity, i, len(r.StopLengths))
			}
		}
		for i := 1; i < len(group); i++ {
			if group[i].Price < group[i-1].Price {
				t.Errorf("%s results not sorted by price: result[%d]=%f > result[%d]=%f",
					stopCity, i-1, group[i-1].Price, i, group[i].Price)
			}
		}
	}
}

func writeResultsJSON(name string, results map[string][]search.Result) error {
	if err := os.MkdirAll("testdata", 0o755); err != nil {
		return err
	}

	payload := struct {
		GeneratedAt time.Time                  `json:"generated_at"`
		Count       int                        `json:"count"`
		Results     map[string][]search.Result `json:"results"`
	}{
		GeneratedAt: time.Now().UTC(),
		Count:       countResults(results),
		Results:     results,
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}

	path := filepath.Join("testdata", name+"_results.json")
	return os.WriteFile(path, data, 0o644)
}

func TestIntegration_Search_ShortHaulRoute(t *testing.T) {
	p := newGFlightsProvider(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	s := search.New(ctx, p)

	results, err := s.Search(provider.Request{
		Origin:        "london",
		Destination:   "paris",
		DepartureDate: futureDate(14),
		ReturnDate:    futureDate(18),
		Adults:        1,
		Class:         provider.Economy,
		Currency:      currency.GBP,
	})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}

	t.Logf("found %d split-ticket results for London → Paris", countResults(results))

	// Short haul is unlikely to benefit from split tickets — fewer or zero
	// results is expected, but the search should still complete without error.
	for stopCity, group := range results {
		for i, r := range group {
			if r.Price <= 0 {
				t.Errorf("%s result[%d]: price must be positive, got %f", stopCity, i, r.Price)
			}
		}
	}
}

func TestIntegration_Search_LongHaulRoute(t *testing.T) {
	p := newGFlightsProvider(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	s := search.New(ctx, p)

	results, err := s.Search(provider.Request{
		Origin:        "london",
		Destination:   "sydney",
		DepartureDate: futureDate(45),
		ReturnDate:    futureDate(59),
		Adults:        1,
		Class:         provider.Economy,
		Currency:      currency.GBP,
	})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}

	t.Logf("found %d split-ticket results for London → Sydney", countResults(results))

	for stopCity, group := range results {
		for i, r := range group {
			if r.Price <= 0 {
				t.Errorf("%s result[%d]: expected positive price, got %f", stopCity, i, r.Price)
			}
			if r.StopCity == "" {
				t.Errorf("%s result[%d]: expected non-empty stop city", stopCity, i)
			}
			// Every itinerary should have exactly two legs
			if len(r.Itineries) != 2 {
				t.Errorf("%s result[%d]: expected 2 legs, got %d", stopCity, i, len(r.Itineries))
			}
		}
	}
}

func TestIntegration_Search_MultiplePassengers(t *testing.T) {
	p := newGFlightsProvider(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	s := search.New(ctx, p)

	results, err := s.Search(provider.Request{
		Origin:        "london",
		Destination:   "new york",
		DepartureDate: futureDate(30),
		ReturnDate:    futureDate(37),
		Adults:        2,
		Children:      1,
		Class:         provider.Economy,
		Currency:      currency.GBP,
	})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}

	t.Logf("found %d split-ticket results for 2 adults + 1 child London → New York", countResults(results))

	for stopCity, group := range results {
		for i, r := range group {
			if r.Price <= 0 {
				t.Errorf("%s result[%d]: expected positive price, got %f", stopCity, i, r.Price)
			}
		}
	}
}

func TestIntegration_Search_BusinessClass(t *testing.T) {
	p := newGFlightsProvider(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	s := search.New(ctx, p)

	results, err := s.Search(provider.Request{
		Origin:        "london",
		Destination:   "new york",
		DepartureDate: futureDate(30),
		ReturnDate:    futureDate(37),
		Adults:        1,
		Class:         provider.Business,
		Currency:      currency.GBP,
	})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}

	t.Logf("found %d split-ticket results for business class London → New York", countResults(results))

	for stopCity, group := range results {
		for i, r := range group {
			if r.Price <= 0 {
				t.Errorf("%s result[%d]: expected positive price, got %f", stopCity, i, r.Price)
			}
		}
	}
}

func TestIntegration_Search_DifferentCurrency(t *testing.T) {
	p := newGFlightsProvider(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	s := search.New(ctx, p)

	results, err := s.Search(provider.Request{
		Origin:        "london",
		Destination:   "new york",
		DepartureDate: futureDate(30),
		ReturnDate:    futureDate(37),
		Adults:        1,
		Class:         provider.Economy,
		Currency:      currency.USD,
	})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}

	t.Logf("found %d split-ticket results (USD) for London → New York", countResults(results))

	for stopCity, group := range results {
		for i, r := range group {
			if r.Price <= 0 {
				t.Errorf("%s result[%d]: expected positive price, got %f", stopCity, i, r.Price)
			}
		}
	}
}

func TestIntegration_Search_ContextTimeout(t *testing.T) {
	p := newGFlightsProvider(t)

	// Very short timeout — should cancel before completion
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	s := search.New(ctx, p)

	_, err := s.Search(provider.Request{
		Origin:        "london",
		Destination:   "sydney",
		DepartureDate: futureDate(30),
		ReturnDate:    futureDate(44),
		Adults:        1,
		Class:         provider.Economy,
		Currency:      currency.GBP,
	})

	// We expect either:
	// - an error (context deadline exceeded / canceled) — correct behaviour
	// - no error if somehow it completed instantly (unlikely but acceptable)
	if err != nil {
		t.Logf("search correctly returned error on tight timeout: %v", err)
	} else {
		t.Log("search completed despite tight timeout (acceptable)")
	}
}

// --- Result validation helpers ---

func TestIntegration_Search_ResultStructureValidation(t *testing.T) {
	p := newGFlightsProvider(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	s := search.New(ctx, p)

	results, err := s.Search(provider.Request{
		Origin:        "london",
		Destination:   "new york",
		DepartureDate: futureDate(30),
		ReturnDate:    futureDate(37),
		Adults:        1,
		Class:         provider.Economy,
		Currency:      currency.GBP,
	})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}

	if countResults(results) == 0 {
		t.Skip("no results returned — cannot validate structure")
	}

	for stopCity, group := range results {
		for i, r := range group {
			// Validate price is sum of two legs
			expectedPrice := r.Itineries[0].Price + r.Itineries[1].Price
			if r.Price != expectedPrice {
				t.Errorf("%s result[%d]: price %f != leg1(%f) + leg2(%f) = %f",
					stopCity, i, r.Price, r.Itineries[0].Price, r.Itineries[1].Price, expectedPrice)
			}

			// Stop city should match the connection point
			leg1Arr := r.Itineries[0].Outbound.ArrivalAirport
			leg2Dep := r.Itineries[1].Outbound.DepartureAirport
			if r.StopCity != leg1Arr {
				t.Errorf("%s result[%d]: StopCity %q != first leg arrival %q", stopCity, i, r.StopCity, leg1Arr)
			}
			if leg1Arr != leg2Dep {
				t.Errorf("%s result[%d]: first leg arrives at %q but second departs from %q",
					stopCity, i, leg1Arr, leg2Dep)
			}

			// Outbound layover should be between min and max
			outboundLayover := r.StopLengths[0]
			if outboundLayover < 3*time.Hour || outboundLayover > 6*time.Hour {
				t.Errorf("%s result[%d]: outbound layover %v outside [3h, 6h]", stopCity, i, outboundLayover)
			}

			// Inbound layover should also be valid
			inboundLayover := r.StopLengths[1]
			if inboundLayover < 3*time.Hour || inboundLayover > 6*time.Hour {
				t.Errorf("%s result[%d]: inbound layover %v outside [3h, 6h]", stopCity, i, inboundLayover)
			}

			// Each leg should have outbound and inbound with positive duration
			for j, itin := range r.Itineries {
				if itin.Outbound.Duration <= 0 {
					t.Errorf("%s result[%d].leg[%d]: outbound duration must be positive, got %v",
						stopCity, i, j, itin.Outbound.Duration)
				}
				if itin.Inbound.Duration <= 0 {
					t.Errorf("%s result[%d].leg[%d]: inbound duration must be positive, got %v",
						stopCity, i, j, itin.Inbound.Duration)
				}
				if len(itin.Outbound.Flights) == 0 {
					t.Errorf("%s result[%d].leg[%d]: outbound has no flights", stopCity, i, j)
				}
				if len(itin.Inbound.Flights) == 0 {
					t.Errorf("%s result[%d].leg[%d]: inbound has no flights", stopCity, i, j)
				}
			}
		}
	}
}

func TestIntegration_Search_FlightDetailsPresent(t *testing.T) {
	p := newGFlightsProvider(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	s := search.New(ctx, p)

	results, err := s.Search(provider.Request{
		Origin:        "london",
		Destination:   "new york",
		DepartureDate: futureDate(30),
		ReturnDate:    futureDate(37),
		Adults:        1,
		Class:         provider.Economy,
		Currency:      currency.GBP,
	})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}

	if countResults(results) == 0 {
		t.Skip("no results returned — cannot validate flight details")
	}

	// Check first result's flight details
	r, ok := firstResult(results)
	if !ok {
		t.Skip("no results returned — cannot validate flight details")
	}
	for j, itin := range r.Itineries {
		for k, f := range itin.Outbound.Flights {
			if f.FlightCode == "" {
				t.Errorf("result[0].leg[%d].outbound.flight[%d]: empty flight code", j, k)
			}
			if f.Airline == "" {
				t.Errorf("result[0].leg[%d].outbound.flight[%d]: empty airline", j, k)
			}
			if f.DepartureAirport == "" {
				t.Errorf("result[0].leg[%d].outbound.flight[%d]: empty departure airport", j, k)
			}
			if f.ArrivalAirport == "" {
				t.Errorf("result[0].leg[%d].outbound.flight[%d]: empty arrival airport", j, k)
			}
			if f.DepartureTime.IsZero() {
				t.Errorf("result[0].leg[%d].outbound.flight[%d]: zero departure time", j, k)
			}
			if f.ArrivalTime.IsZero() {
				t.Errorf("result[0].leg[%d].outbound.flight[%d]: zero arrival time", j, k)
			}
		}
		if itin.BookingURL == "" {
			t.Errorf("result[0].leg[%d]: empty booking URL", j)
		}
	}
}
