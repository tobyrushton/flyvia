package search

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	"golang.org/x/text/currency"

	"github.com/tobyrushton/flyvia/packages/search/itinery"
	"github.com/tobyrushton/flyvia/packages/search/leg"
	"github.com/tobyrushton/flyvia/packages/search/provider"
	"github.com/tobyrushton/flyvia/packages/search/provider/providerfakes"
)

// --- Helpers ---

var baseTime = time.Date(2025, 6, 1, 8, 0, 0, 0, time.UTC)

func makeLeg(dep, arr string, depTime, arrTime time.Time, stops int) leg.Leg {
	return leg.Leg{
		Flights: []leg.Flight{{
			DepartureTime:    depTime,
			ArrivalTime:      arrTime,
			DepartureAirport: dep,
			ArrivalAirport:   arr,
			FlightCode:       "XX100",
			Plane:            "A320",
			Airline:          "MockAir",
		}},
		Stops:            stops,
		DepartureTime:    depTime,
		ArrivalTime:      arrTime,
		DepartureAirport: dep,
		ArrivalAirport:   arr,
		Duration:         arrTime.Sub(depTime),
	}
}

func makeItin(depAirport, arrAirport string, depTime, arrTime time.Time, price float64) itinery.Itinery {
	returnDep := depTime.Add(7 * 24 * time.Hour)
	returnArr := arrTime.Add(7 * 24 * time.Hour)
	return itinery.Itinery{
		Outbound:   makeLeg(depAirport, arrAirport, depTime, arrTime, 0),
		Inbound:    makeLeg(arrAirport, depAirport, returnDep, returnArr, 0),
		Price:      price,
		BookingURL: "https://example.com/book",
	}
}

func defaultRequest() provider.Request {
	return provider.Request{
		Origin:        "LHR",
		Destination:   "LAX",
		DepartureDate: baseTime,
		ReturnDate:    baseTime.Add(7 * 24 * time.Hour),
		Adults:        1,
		Children:      0,
		Class:         provider.Economy,
		Currency:      currency.GBP,
	}
}

type searchKey struct {
	Origin      string
	Destination string
}

type exploreKey struct {
	Origin string
}

// setupFake configures a FakeProvider with map-based dispatch for Search and Explore,
// plus a real sort-by-price implementation.
func setupFake(
	searchResults map[searchKey][]itinery.Itinery,
	searchErrors map[searchKey]error,
	exploreResults map[exploreKey][]itinery.ExploreItinery,
	exploreErrors map[exploreKey]error,
) *providerfakes.FakeProvider {
	fake := &providerfakes.FakeProvider{}

	fake.SearchCalls(func(_ context.Context, req provider.Request) ([]itinery.Itinery, error) {
		key := searchKey{Origin: req.Origin, Destination: req.Destination}
		if err, ok := searchErrors[key]; ok {
			return nil, err
		}
		if res, ok := searchResults[key]; ok {
			return res, nil
		}
		return []itinery.Itinery{}, nil
	})

	fake.ExploreCalls(func(_ context.Context, _ provider.Request, origin string) ([]itinery.ExploreItinery, error) {
		key := exploreKey{Origin: origin}
		if err, ok := exploreErrors[key]; ok {
			return nil, err
		}
		if res, ok := exploreResults[key]; ok {
			return res, nil
		}
		return []itinery.ExploreItinery{}, nil
	})

	fake.SortByPriceCalls(func(itins *[]itinery.Itinery) {
		sort.Slice(*itins, func(i, j int) bool {
			return (*itins)[i].Price < (*itins)[j].Price
		})
	})

	return fake
}

// --- Tests ---

func TestNew(t *testing.T) {
	ctx := context.Background()
	fake := setupFake(nil, nil, nil, nil)

	s := New(ctx, fake)

	if s == nil {
		t.Fatal("expected non-nil Search")
	}
	if s.ctx != ctx {
		t.Error("context not set correctly")
	}
}

func TestSearch_BasePriceError(t *testing.T) {
	req := defaultRequest()
	fake := setupFake(
		nil,
		map[searchKey]error{
			{Origin: req.Origin, Destination: req.Destination}: errors.New("search failed"),
		},
		nil, nil,
	)

	s := New(context.Background(), fake)
	_, err := s.Search(req)

	if err == nil {
		t.Fatal("expected error from base price search")
	}
	if err.Error() != "search failed" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSearch_ExploreOriginError(t *testing.T) {
	req := defaultRequest()
	fake := setupFake(
		map[searchKey][]itinery.Itinery{
			{Origin: req.Origin, Destination: req.Destination}: {},
		},
		nil,
		nil,
		map[exploreKey]error{
			{Origin: req.Destination}: errors.New("explore origin error"),
		},
	)

	s := New(context.Background(), fake)
	_, err := s.Search(req)

	if err == nil {
		t.Fatal("expected error from explore")
	}
}

func TestSearch_ExploreDestinationError(t *testing.T) {
	req := defaultRequest()
	fake := setupFake(
		map[searchKey][]itinery.Itinery{
			{Origin: req.Origin, Destination: req.Destination}: {},
		},
		nil,
		nil,
		map[exploreKey]error{
			{Origin: req.Origin}: errors.New("explore dest error"),
		},
	)

	s := New(context.Background(), fake)
	_, err := s.Search(req)

	if err == nil {
		t.Fatal("expected error from explore")
	}
}

func TestSearch_EmptyBasePrice_NoExploreResults(t *testing.T) {
	req := defaultRequest()
	fake := setupFake(
		map[searchKey][]itinery.Itinery{
			{Origin: req.Origin, Destination: req.Destination}: {},
		},
		nil, nil, nil,
	)

	s := New(context.Background(), fake)
	results, err := s.Search(req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestSearch_FilterReasonableItineraries_AllFiltered(t *testing.T) {
	req := defaultRequest()
	baseItin := makeItin("LHR", "LAX", baseTime, baseTime.Add(11*time.Hour), 200.0)
	fake := setupFake(
		map[searchKey][]itinery.Itinery{
			{Origin: req.Origin, Destination: req.Destination}: {baseItin},
		},
		nil,
		map[exploreKey][]itinery.ExploreItinery{
			{Origin: req.Destination}: {
				{Destination: "JFK", Price: 300.0},
				{Destination: "CDG", Price: 250.0},
			},
			{Origin: req.Origin}: {
				{Destination: "DXB", Price: 400.0},
			},
		},
		nil,
	)

	s := New(context.Background(), fake)
	results, err := s.Search(req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results after filtering, got %d", len(results))
	}
}

func TestSearch_ExpandFirstLegsError(t *testing.T) {
	req := defaultRequest()
	baseItin := makeItin("LHR", "LAX", baseTime, baseTime.Add(11*time.Hour), 500.0)
	fake := setupFake(
		map[searchKey][]itinery.Itinery{
			{Origin: req.Origin, Destination: req.Destination}: {baseItin},
		},
		map[searchKey]error{
			{Origin: "LHR", Destination: "JFK"}: errors.New("expand first leg failed"),
		},
		map[exploreKey][]itinery.ExploreItinery{
			{Origin: req.Destination}: {{Destination: "JFK", Price: 50.0}},
			{Origin: req.Origin}:      {},
		},
		nil,
	)

	s := New(context.Background(), fake)
	_, err := s.Search(req)

	if err == nil {
		t.Fatal("expected error from expand first legs")
	}
}

func TestSearch_ExpandSecondLegsError(t *testing.T) {
	req := defaultRequest()
	baseItin := makeItin("LHR", "LAX", baseTime, baseTime.Add(11*time.Hour), 500.0)
	fake := setupFake(
		map[searchKey][]itinery.Itinery{
			{Origin: req.Origin, Destination: req.Destination}: {baseItin},
			{Origin: "LHR", Destination: "JFK"}: {
				makeItin("LHR", "JFK", baseTime, baseTime.Add(8*time.Hour), 200.0),
			},
		},
		map[searchKey]error{
			{Origin: "JFK", Destination: "LAX"}: errors.New("expand second leg failed"),
		},
		map[exploreKey][]itinery.ExploreItinery{
			{Origin: req.Destination}: {{Destination: "JFK", Price: 50.0}},
			{Origin: req.Origin}:      {},
		},
		nil,
	)

	s := New(context.Background(), fake)
	_, err := s.Search(req)

	if err == nil {
		t.Fatal("expected error from expand second legs")
	}
}

func TestSearch_EndToEnd_ValidCombination(t *testing.T) {
	req := defaultRequest()
	baseItin := makeItin("LHR", "LAX", baseTime, baseTime.Add(11*time.Hour), 2000.0)
	firstLeg := itinery.Itinery{
		Outbound: makeLeg("LHR", "JFK", baseTime, baseTime.Add(8*time.Hour), 0),
		Inbound:  makeLeg("JFK", "LHR", baseTime.Add(7*24*time.Hour), baseTime.Add(7*24*time.Hour+8*time.Hour), 0),
		Price:    300.0,
	}
	secondLeg := itinery.Itinery{
		Outbound: makeLeg("JFK", "LAX", baseTime.Add(12*time.Hour), baseTime.Add(17*time.Hour), 0),
		Inbound:  makeLeg("LAX", "JFK", baseTime.Add(7*24*time.Hour-3*time.Hour), baseTime.Add(7*24*time.Hour-1*time.Hour), 0),
		Price:    200.0,
	}
	fake := setupFake(
		map[searchKey][]itinery.Itinery{
			{Origin: req.Origin, Destination: req.Destination}: {baseItin},
			{Origin: "LHR", Destination: "JFK"}:                {firstLeg},
			{Origin: "JFK", Destination: "LAX"}:                {secondLeg},
		},
		nil,
		map[exploreKey][]itinery.ExploreItinery{
			{Origin: req.Destination}: {{Destination: "JFK", Price: 100.0}},
			{Origin: req.Origin}:      {},
		},
		nil,
	)

	s := New(context.Background(), fake)
	results, err := s.Search(req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Price != 500.0 {
		t.Errorf("expected combined price 500.0, got %f", r.Price)
	}
	if r.StopCity != "JFK" {
		t.Errorf("expected stop city JFK, got %s", r.StopCity)
	}
}

func TestSearch_EndToEnd_NoValidLayover(t *testing.T) {
	req := defaultRequest()
	baseItin := makeItin("LHR", "LAX", baseTime, baseTime.Add(11*time.Hour), 1000.0)
	firstLeg := itinery.Itinery{
		Outbound: makeLeg("LHR", "JFK", baseTime, baseTime.Add(8*time.Hour), 0),
		Inbound:  makeLeg("JFK", "LHR", baseTime.Add(7*24*time.Hour), baseTime.Add(7*24*time.Hour+8*time.Hour), 0),
		Price:    300.0,
	}
	secondLeg := itinery.Itinery{
		Outbound: makeLeg("JFK", "LAX", baseTime.Add(8*time.Hour+30*time.Minute), baseTime.Add(13*time.Hour), 0),
		Inbound:  makeLeg("LAX", "JFK", baseTime.Add(7*24*time.Hour-3*time.Hour), baseTime.Add(7*24*time.Hour-1*time.Hour), 0),
		Price:    200.0,
	}
	fake := setupFake(
		map[searchKey][]itinery.Itinery{
			{Origin: req.Origin, Destination: req.Destination}: {baseItin},
			{Origin: "LHR", Destination: "JFK"}:                {firstLeg},
			{Origin: "JFK", Destination: "LAX"}:                {secondLeg},
		},
		nil,
		map[exploreKey][]itinery.ExploreItinery{
			{Origin: req.Destination}: {{Destination: "JFK", Price: 100.0}},
			{Origin: req.Origin}:      {},
		},
		nil,
	)

	s := New(context.Background(), fake)
	results, err := s.Search(req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results with invalid layover, got %d", len(results))
	}
}

func TestSearch_EndToEnd_MultipleStops(t *testing.T) {
	req := defaultRequest()
	baseItin := makeItin("LHR", "LAX", baseTime, baseTime.Add(11*time.Hour), 1000.0)
	fake := setupFake(
		map[searchKey][]itinery.Itinery{
			{Origin: req.Origin, Destination: req.Destination}: {baseItin},
			{Origin: "LHR", Destination: "JFK"}: {
				{
					Outbound: makeLeg("LHR", "JFK", baseTime, baseTime.Add(8*time.Hour), 0),
					Inbound:  makeLeg("JFK", "LHR", baseTime.Add(7*24*time.Hour), baseTime.Add(7*24*time.Hour+8*time.Hour), 0),
					Price:    300.0,
				},
			},
			{Origin: "JFK", Destination: "LAX"}: {
				{
					Outbound: makeLeg("JFK", "LAX", baseTime.Add(12*time.Hour), baseTime.Add(17*time.Hour), 0),
					Inbound:  makeLeg("LAX", "JFK", baseTime.Add(7*24*time.Hour-3*time.Hour), baseTime.Add(7*24*time.Hour-1*time.Hour), 0),
					Price:    200.0,
				},
			},
			{Origin: "LHR", Destination: "ORD"}: {
				{
					Outbound: makeLeg("LHR", "ORD", baseTime, baseTime.Add(9*time.Hour), 0),
					Inbound:  makeLeg("ORD", "LHR", baseTime.Add(7*24*time.Hour), baseTime.Add(7*24*time.Hour+9*time.Hour), 0),
					Price:    280.0,
				},
			},
			{Origin: "ORD", Destination: "LAX"}: {
				{
					Outbound: makeLeg("ORD", "LAX", baseTime.Add(13*time.Hour), baseTime.Add(16*time.Hour), 0),
					Inbound:  makeLeg("LAX", "ORD", baseTime.Add(7*24*time.Hour-4*time.Hour), baseTime.Add(7*24*time.Hour-1*time.Hour), 0),
					Price:    150.0,
				},
			},
		},
		nil,
		map[exploreKey][]itinery.ExploreItinery{
			{Origin: req.Destination}: {
				{Destination: "JFK", Price: 100.0},
				{Destination: "ORD", Price: 120.0},
			},
			{Origin: req.Origin}: {},
		},
		nil,
	)

	s := New(context.Background(), fake)
	results, err := s.Search(req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results for two stops, got %d", len(results))
	}
	// Should be sorted by price: ORD stop (280+150=430) then JFK (300+200=500)
	if results[0].Price != 430.0 {
		t.Errorf("first result price expected 430.0, got %f", results[0].Price)
	}
	if results[1].Price != 500.0 {
		t.Errorf("second result price expected 500.0, got %f", results[1].Price)
	}
}

func TestSearch_ResultsSortedByPrice(t *testing.T) {
	req := defaultRequest()
	baseItin := makeItin("LHR", "LAX", baseTime, baseTime.Add(11*time.Hour), 2000.0)
	fake := setupFake(
		map[searchKey][]itinery.Itinery{
			{Origin: req.Origin, Destination: req.Destination}: {baseItin},
			{Origin: "LHR", Destination: "JFK"}: {
				{
					Outbound: makeLeg("LHR", "JFK", baseTime, baseTime.Add(8*time.Hour), 0),
					Inbound:  makeLeg("JFK", "LHR", baseTime.Add(7*24*time.Hour), baseTime.Add(7*24*time.Hour+8*time.Hour), 0),
					Price:    600.0,
				},
				{
					Outbound: makeLeg("LHR", "JFK", baseTime.Add(1*time.Hour), baseTime.Add(9*time.Hour), 0),
					Inbound:  makeLeg("JFK", "LHR", baseTime.Add(7*24*time.Hour+1*time.Hour), baseTime.Add(7*24*time.Hour+9*time.Hour), 0),
					Price:    300.0,
				},
			},
			{Origin: "JFK", Destination: "LAX"}: {
				{
					Outbound: makeLeg("JFK", "LAX", baseTime.Add(12*time.Hour), baseTime.Add(17*time.Hour), 0),
					Inbound:  makeLeg("LAX", "JFK", baseTime.Add(7*24*time.Hour-3*time.Hour), baseTime.Add(7*24*time.Hour-1*time.Hour), 0),
					Price:    400.0,
				},
			},
		},
		nil,
		map[exploreKey][]itinery.ExploreItinery{
			{Origin: req.Destination}: {{Destination: "JFK", Price: 50.0}},
			{Origin: req.Origin}:      {},
		},
		nil,
	)

	s := New(context.Background(), fake)
	results, err := s.Search(req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i := 1; i < len(results); i++ {
		if results[i].Price < results[i-1].Price {
			t.Errorf("results not sorted by price: %f before %f", results[i-1].Price, results[i].Price)
		}
	}
}

func TestSearch_BothExploreDirections(t *testing.T) {
	req := defaultRequest()
	baseItin := makeItin("LHR", "LAX", baseTime, baseTime.Add(11*time.Hour), 2000.0)
	fake := setupFake(
		map[searchKey][]itinery.Itinery{
			{Origin: req.Origin, Destination: req.Destination}: {baseItin},
			{Origin: "LHR", Destination: "JFK"}: {
				{
					Outbound: makeLeg("LHR", "JFK", baseTime, baseTime.Add(8*time.Hour), 0),
					Inbound:  makeLeg("JFK", "LHR", baseTime.Add(7*24*time.Hour), baseTime.Add(7*24*time.Hour+8*time.Hour), 0),
					Price:    300.0,
				},
			},
			{Origin: "JFK", Destination: "LAX"}: {
				{
					Outbound: makeLeg("JFK", "LAX", baseTime.Add(12*time.Hour), baseTime.Add(17*time.Hour), 0),
					Inbound:  makeLeg("LAX", "JFK", baseTime.Add(7*24*time.Hour-3*time.Hour), baseTime.Add(7*24*time.Hour-1*time.Hour), 0),
					Price:    200.0,
				},
			},
			{Origin: "LHR", Destination: "DUB"}: {
				{
					Outbound: makeLeg("LHR", "DUB", baseTime, baseTime.Add(1*time.Hour), 0),
					Inbound:  makeLeg("DUB", "LHR", baseTime.Add(7*24*time.Hour), baseTime.Add(7*24*time.Hour+1*time.Hour), 0),
					Price:    100.0,
				},
			},
		},
		nil,
		map[exploreKey][]itinery.ExploreItinery{
			{Origin: req.Destination}: {{Destination: "JFK", Price: 50.0}},
			{Origin: req.Origin}:      {{Destination: "DUB", Price: 30.0}},
		},
		nil,
	)

	s := New(context.Background(), fake)
	results, err := s.Search(req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) == 0 {
		t.Error("expected at least one result from both explore directions")
	}
}

// --- filterReasonableItineraries tests ---

func TestFilterReasonableItineraries_AllPass(t *testing.T) {
	s := &Search{}
	itins := []itinery.ExploreItinery{
		{Destination: "JFK", Price: 50.0},
		{Destination: "CDG", Price: 100.0},
		{Destination: "DXB", Price: 150.0},
	}

	result := s.filterReasonableItineraries(itins, 1000.0)

	if len(result) != 3 {
		t.Errorf("expected 3 itineraries to pass, got %d", len(result))
	}
}

func TestFilterReasonableItineraries_NonePass(t *testing.T) {
	s := &Search{}
	itins := []itinery.ExploreItinery{
		{Destination: "JFK", Price: 900.0},
		{Destination: "CDG", Price: 850.0},
	}
	// basePrice * 0.8 = 800 -> all are > 800

	result := s.filterReasonableItineraries(itins, 1000.0)

	if len(result) != 0 {
		t.Errorf("expected 0 itineraries to pass, got %d", len(result))
	}
}

func TestFilterReasonableItineraries_SomePass(t *testing.T) {
	s := &Search{}
	itins := []itinery.ExploreItinery{
		{Destination: "JFK", Price: 100.0}, // passes (100 < 400)
		{Destination: "CDG", Price: 450.0}, // fails (450 > 400)
		{Destination: "DXB", Price: 399.0}, // passes (399 < 400)
		{Destination: "SIN", Price: 400.0}, // passes (400 > 400 is false, not filtered)
		{Destination: "NRT", Price: 401.0}, // fails (401 > 400)
	}
	// basePrice = 500, basePrice * 0.8 = 400

	result := s.filterReasonableItineraries(itins, 500.0)

	if len(result) != 3 {
		t.Errorf("expected 3 itineraries to pass, got %d", len(result))
	}
}

func TestFilterReasonableItineraries_ExactBoundary(t *testing.T) {
	s := &Search{}
	itins := []itinery.ExploreItinery{
		{Destination: "JFK", Price: 80.0}, // exactly at basePrice * 0.8
	}
	// basePrice = 100, threshold = 80. Price > 80 -> false (80 is not > 80), so it passes

	result := s.filterReasonableItineraries(itins, 100.0)

	if len(result) != 1 {
		t.Errorf("expected 1 itinerary at exact boundary to pass, got %d", len(result))
	}
}

func TestFilterReasonableItineraries_ZeroBasePrice(t *testing.T) {
	s := &Search{}
	itins := []itinery.ExploreItinery{
		{Destination: "JFK", Price: 50.0},
	}
	// basePrice = 0, threshold = 0. Price 50 > 0 -> filtered out

	result := s.filterReasonableItineraries(itins, 0.0)

	if len(result) != 0 {
		t.Errorf("expected 0 itineraries with zero base price, got %d", len(result))
	}
}

func TestFilterReasonableItineraries_Empty(t *testing.T) {
	s := &Search{}

	result := s.filterReasonableItineraries([]itinery.ExploreItinery{}, 1000.0)

	if len(result) != 0 {
		t.Errorf("expected 0 itineraries from empty input, got %d", len(result))
	}
}

// --- getBasePrice tests ---

func TestGetBasePrice_EmptyOffers(t *testing.T) {
	req := defaultRequest()
	fake := setupFake(
		map[searchKey][]itinery.Itinery{
			{Origin: req.Origin, Destination: req.Destination}: {},
		},
		nil, nil, nil,
	)

	s := New(context.Background(), fake)
	price, err := s.getBasePrice(req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if price != 0 {
		t.Errorf("expected 0 base price for empty offers, got %f", price)
	}
}

func TestGetBasePrice_SearchError(t *testing.T) {
	req := defaultRequest()
	fake := setupFake(
		nil,
		map[searchKey]error{
			{Origin: req.Origin, Destination: req.Destination}: errors.New("search error"),
		},
		nil, nil,
	)

	s := New(context.Background(), fake)
	_, err := s.getBasePrice(req)

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetBasePrice_SingleOffer(t *testing.T) {
	req := defaultRequest()
	itin := itinery.Itinery{
		Outbound: makeLeg("LHR", "LAX", baseTime, baseTime.Add(11*time.Hour), 0),
		Inbound:  makeLeg("LAX", "LHR", baseTime.Add(7*24*time.Hour), baseTime.Add(7*24*time.Hour+11*time.Hour), 0),
		Price:    500.0,
	}
	fake := setupFake(
		map[searchKey][]itinery.Itinery{
			{Origin: req.Origin, Destination: req.Destination}: {itin},
		},
		nil, nil, nil,
	)

	s := New(context.Background(), fake)
	price, err := s.getBasePrice(req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if price <= 0 {
		t.Errorf("expected positive base price, got %f", price)
	}
}

func TestGetBasePrice_MultipleOffers(t *testing.T) {
	req := defaultRequest()
	itins := []itinery.Itinery{
		{
			Outbound: makeLeg("LHR", "LAX", baseTime, baseTime.Add(11*time.Hour), 0),
			Inbound:  makeLeg("LAX", "LHR", baseTime.Add(7*24*time.Hour), baseTime.Add(7*24*time.Hour+11*time.Hour), 0),
			Price:    300.0,
		},
		{
			Outbound: makeLeg("LHR", "LAX", baseTime, baseTime.Add(12*time.Hour), 1),
			Inbound:  makeLeg("LAX", "LHR", baseTime.Add(7*24*time.Hour), baseTime.Add(7*24*time.Hour+12*time.Hour), 1),
			Price:    500.0,
		},
		{
			Outbound: makeLeg("LHR", "LAX", baseTime, baseTime.Add(14*time.Hour), 2),
			Inbound:  makeLeg("LAX", "LHR", baseTime.Add(7*24*time.Hour), baseTime.Add(7*24*time.Hour+14*time.Hour), 2),
			Price:    800.0,
		},
	}
	fake := setupFake(
		map[searchKey][]itinery.Itinery{
			{Origin: req.Origin, Destination: req.Destination}: itins,
		},
		nil, nil, nil,
	)

	s := New(context.Background(), fake)
	price, err := s.getBasePrice(req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if price < 370.0 {
		t.Errorf("expected base price > 370 (blended minimum), got %f", price)
	}
}

func TestGetBasePrice_QualityAdjustment_DirectFlight(t *testing.T) {
	req := defaultRequest()
	itin := itinery.Itinery{
		Outbound: makeLeg("LHR", "CDG", baseTime, baseTime.Add(1*time.Hour), 0),
		Inbound:  makeLeg("CDG", "LHR", baseTime.Add(7*24*time.Hour), baseTime.Add(7*24*time.Hour+1*time.Hour), 0),
		Price:    100.0,
	}
	fake := setupFake(
		map[searchKey][]itinery.Itinery{
			{Origin: req.Origin, Destination: req.Destination}: {itin},
		},
		nil, nil, nil,
	)

	s := New(context.Background(), fake)
	price, err := s.getBasePrice(req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if price > 120.0 {
		t.Errorf("expected low quality adjustment for direct flight, got price %f", price)
	}
}

func TestGetBasePrice_QualityAdjustment_LongMultiStop(t *testing.T) {
	req := defaultRequest()
	itin := itinery.Itinery{
		Outbound: makeLeg("LHR", "SYD", baseTime, baseTime.Add(24*time.Hour), 3),
		Inbound:  makeLeg("SYD", "LHR", baseTime.Add(7*24*time.Hour), baseTime.Add(8*24*time.Hour), 3),
		Price:    1000.0,
	}
	fake := setupFake(
		map[searchKey][]itinery.Itinery{
			{Origin: req.Origin, Destination: req.Destination}: {itin},
		},
		nil, nil, nil,
	)

	s := New(context.Background(), fake)
	price, err := s.getBasePrice(req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if price <= 1000.0 {
		t.Errorf("expected quality adjustment to increase base price above 1000, got %f", price)
	}
}

// --- explore tests ---

func TestExplore_BothSucceed(t *testing.T) {
	req := defaultRequest()
	fake := setupFake(
		nil, nil,
		map[exploreKey][]itinery.ExploreItinery{
			{Origin: req.Destination}: {
				{Destination: "JFK", Price: 200.0},
				{Destination: "CDG", Price: 100.0},
			},
			{Origin: req.Origin}: {
				{Destination: "DUB", Price: 50.0},
			},
		},
		nil,
	)

	s := New(context.Background(), fake)
	exploreOr, exploreDest, err := s.explore(req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(exploreOr) != 2 {
		t.Errorf("expected 2 origin explore results, got %d", len(exploreOr))
	}
	if len(exploreDest) != 1 {
		t.Errorf("expected 1 dest explore results, got %d", len(exploreDest))
	}
	if exploreOr[0].Price > exploreOr[1].Price {
		t.Error("exploreOr not sorted by price")
	}
}

func TestExplore_OriginError(t *testing.T) {
	req := defaultRequest()
	fake := setupFake(
		nil, nil,
		map[exploreKey][]itinery.ExploreItinery{
			{Origin: req.Origin}: {},
		},
		map[exploreKey]error{
			{Origin: req.Destination}: errors.New("explore failed"),
		},
	)

	s := New(context.Background(), fake)
	_, _, err := s.explore(req)

	if err == nil {
		t.Fatal("expected error from explore")
	}
}

func TestExplore_DestError(t *testing.T) {
	req := defaultRequest()
	fake := setupFake(
		nil, nil,
		map[exploreKey][]itinery.ExploreItinery{
			{Origin: req.Destination}: {},
		},
		map[exploreKey]error{
			{Origin: req.Origin}: errors.New("explore dest failed"),
		},
	)

	s := New(context.Background(), fake)
	_, _, err := s.explore(req)

	if err == nil {
		t.Fatal("expected error from explore")
	}
}

func TestExplore_EmptyResults(t *testing.T) {
	req := defaultRequest()
	fake := setupFake(
		nil, nil,
		map[exploreKey][]itinery.ExploreItinery{
			{Origin: req.Destination}: {},
			{Origin: req.Origin}:      {},
		},
		nil,
	)

	s := New(context.Background(), fake)
	exploreOr, exploreDest, err := s.explore(req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(exploreOr) != 0 {
		t.Errorf("expected 0 origin explore results, got %d", len(exploreOr))
	}
	if len(exploreDest) != 0 {
		t.Errorf("expected 0 dest explore results, got %d", len(exploreDest))
	}
}

// --- expandFirstLegs tests ---

func TestExpandFirstLegs_Success(t *testing.T) {
	req := defaultRequest()
	exploreItins := []itinery.ExploreItinery{
		{Destination: "JFK", Price: 100.0},
		{Destination: "CDG", Price: 150.0},
	}
	fake := setupFake(
		map[searchKey][]itinery.Itinery{
			{Origin: "LHR", Destination: "JFK"}: {
				makeItin("LHR", "JFK", baseTime, baseTime.Add(8*time.Hour), 400.0),
				makeItin("LHR", "JFK", baseTime.Add(2*time.Hour), baseTime.Add(10*time.Hour), 300.0),
			},
			{Origin: "LHR", Destination: "CDG"}: {
				makeItin("LHR", "CDG", baseTime, baseTime.Add(1*time.Hour), 100.0),
			},
		},
		nil, nil, nil,
	)

	s := New(context.Background(), fake)
	result, err := s.expandFirstLegs(req, exploreItins)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(result))
	}
	if len(result[0]) == 2 && result[0][0].Price > result[0][1].Price {
		t.Error("JFK results not sorted by price")
	}
}

func TestExpandFirstLegs_Error(t *testing.T) {
	req := defaultRequest()
	exploreItins := []itinery.ExploreItinery{
		{Destination: "JFK", Price: 100.0},
	}
	fake := setupFake(
		nil,
		map[searchKey]error{
			{Origin: "LHR", Destination: "JFK"}: errors.New("search failed"),
		},
		nil, nil,
	)

	s := New(context.Background(), fake)
	_, err := s.expandFirstLegs(req, exploreItins)

	if err == nil {
		t.Fatal("expected error from expandFirstLegs")
	}
}

func TestExpandFirstLegs_Empty(t *testing.T) {
	fake := setupFake(nil, nil, nil, nil)
	req := defaultRequest()

	s := New(context.Background(), fake)
	result, err := s.expandFirstLegs(req, []itinery.ExploreItinery{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 groups, got %d", len(result))
	}
}

// --- expandSecondLegs tests ---

func TestExpandSecondLegs_Success(t *testing.T) {
	req := defaultRequest()
	exploreOr := []itinery.ExploreItinery{
		{Destination: "JFK", Price: 100.0},
	}
	exploreDest := []itinery.ExploreItinery{
		{Destination: "DUB", Price: 50.0},
	}
	fake := setupFake(
		map[searchKey][]itinery.Itinery{
			{Origin: "JFK", Destination: "LAX"}: {
				makeItin("JFK", "LAX", baseTime.Add(12*time.Hour), baseTime.Add(17*time.Hour), 200.0),
			},
			{Origin: "LHR", Destination: "DUB"}: {
				makeItin("LHR", "DUB", baseTime, baseTime.Add(1*time.Hour), 80.0),
			},
		},
		nil, nil, nil,
	)

	s := New(context.Background(), fake)
	secondOr, secondDest, err := s.expandSecondLegs(req, exploreOr, exploreDest)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(secondOr) != 1 {
		t.Errorf("expected 1 secondOr group, got %d", len(secondOr))
	}
	if len(secondDest) != 1 {
		t.Errorf("expected 1 secondDest group, got %d", len(secondDest))
	}
}

func TestExpandSecondLegs_Error(t *testing.T) {
	req := defaultRequest()
	exploreOr := []itinery.ExploreItinery{
		{Destination: "JFK", Price: 100.0},
	}
	fake := setupFake(
		nil,
		map[searchKey]error{
			{Origin: "JFK", Destination: "LAX"}: errors.New("second leg failed"),
		},
		nil, nil,
	)

	s := New(context.Background(), fake)
	_, _, err := s.expandSecondLegs(req, exploreOr, []itinery.ExploreItinery{})

	if err == nil {
		t.Fatal("expected error from expandSecondLegs")
	}
}

// --- combineItineraries tests ---

func TestCombineItineraries_ValidPair(t *testing.T) {
	s := &Search{}
	first := [][]itinery.Itinery{{
		{
			Outbound: makeLeg("LHR", "JFK", baseTime, baseTime.Add(8*time.Hour), 0),
			Inbound:  makeLeg("JFK", "LHR", baseTime.Add(7*24*time.Hour), baseTime.Add(7*24*time.Hour+8*time.Hour), 0),
			Price:    300.0,
		},
	}}
	second := [][]itinery.Itinery{{
		{
			Outbound: makeLeg("JFK", "LAX", baseTime.Add(12*time.Hour), baseTime.Add(17*time.Hour), 0),
			Inbound:  makeLeg("LAX", "JFK", baseTime.Add(7*24*time.Hour-3*time.Hour), baseTime.Add(7*24*time.Hour-1*time.Hour), 0),
			Price:    200.0,
		},
	}}

	results, err := s.combineItineraries(first, second, nil, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Price != 500.0 {
		t.Errorf("expected price 500.0, got %f", results[0].Price)
	}
}

func TestCombineItineraries_NoMatches(t *testing.T) {
	s := &Search{}
	first := [][]itinery.Itinery{{
		{
			Outbound: makeLeg("LHR", "JFK", baseTime, baseTime.Add(8*time.Hour), 0),
			Inbound:  makeLeg("JFK", "LHR", baseTime.Add(7*24*time.Hour), baseTime.Add(7*24*time.Hour+8*time.Hour), 0),
			Price:    300.0,
		},
	}}
	second := [][]itinery.Itinery{{
		{
			Outbound: makeLeg("CDG", "LAX", baseTime.Add(12*time.Hour), baseTime.Add(17*time.Hour), 0),
			Inbound:  makeLeg("LAX", "CDG", baseTime.Add(7*24*time.Hour-3*time.Hour), baseTime.Add(7*24*time.Hour-1*time.Hour), 0),
			Price:    200.0,
		},
	}}

	results, err := s.combineItineraries(first, second, nil, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results with non-matching airports, got %d", len(results))
	}
}

func TestCombineItineraries_EmptyInput(t *testing.T) {
	s := &Search{}

	results, err := s.combineItineraries(nil, nil, nil, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results from empty input, got %d", len(results))
	}
}

func TestCombineItineraries_SortedByPrice(t *testing.T) {
	s := &Search{}
	first := [][]itinery.Itinery{{
		{
			Outbound: makeLeg("LHR", "JFK", baseTime, baseTime.Add(8*time.Hour), 0),
			Inbound:  makeLeg("JFK", "LHR", baseTime.Add(7*24*time.Hour), baseTime.Add(7*24*time.Hour+8*time.Hour), 0),
			Price:    600.0,
		},
		{
			Outbound: makeLeg("LHR", "JFK", baseTime.Add(1*time.Hour), baseTime.Add(9*time.Hour), 0),
			Inbound:  makeLeg("JFK", "LHR", baseTime.Add(7*24*time.Hour+1*time.Hour), baseTime.Add(7*24*time.Hour+9*time.Hour), 0),
			Price:    200.0,
		},
	}}
	second := [][]itinery.Itinery{{
		{
			Outbound: makeLeg("JFK", "LAX", baseTime.Add(12*time.Hour), baseTime.Add(17*time.Hour), 0),
			Inbound:  makeLeg("LAX", "JFK", baseTime.Add(7*24*time.Hour-3*time.Hour), baseTime.Add(7*24*time.Hour-1*time.Hour), 0),
			Price:    300.0,
		},
	}}

	results, err := s.combineItineraries(first, second, nil, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i := 1; i < len(results); i++ {
		if results[i].Price < results[i-1].Price {
			t.Errorf("results not sorted: %f before %f", results[i-1].Price, results[i].Price)
		}
	}
}

// Test with canceled context
func TestSearch_CanceledContext(t *testing.T) {
	req := defaultRequest()
	fake := setupFake(
		map[searchKey][]itinery.Itinery{
			{Origin: req.Origin, Destination: req.Destination}: {},
		},
		nil, nil, nil,
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s := New(ctx, fake)
	_, err := s.Search(req)

	if err != nil {
		t.Logf("search with canceled context returned error: %v", err)
	}
}

// Test that counterfeiter tracks call counts correctly
func TestSearch_VerifyCallCounts(t *testing.T) {
	req := defaultRequest()
	baseItin := makeItin("LHR", "LAX", baseTime, baseTime.Add(11*time.Hour), 2000.0)
	firstLeg := itinery.Itinery{
		Outbound: makeLeg("LHR", "JFK", baseTime, baseTime.Add(8*time.Hour), 0),
		Inbound:  makeLeg("JFK", "LHR", baseTime.Add(7*24*time.Hour), baseTime.Add(7*24*time.Hour+8*time.Hour), 0),
		Price:    300.0,
	}
	secondLeg := itinery.Itinery{
		Outbound: makeLeg("JFK", "LAX", baseTime.Add(12*time.Hour), baseTime.Add(17*time.Hour), 0),
		Inbound:  makeLeg("LAX", "JFK", baseTime.Add(7*24*time.Hour-3*time.Hour), baseTime.Add(7*24*time.Hour-1*time.Hour), 0),
		Price:    200.0,
	}
	fake := setupFake(
		map[searchKey][]itinery.Itinery{
			{Origin: req.Origin, Destination: req.Destination}: {baseItin},
			{Origin: "LHR", Destination: "JFK"}:                {firstLeg},
			{Origin: "JFK", Destination: "LAX"}:                {secondLeg},
		},
		nil,
		map[exploreKey][]itinery.ExploreItinery{
			{Origin: req.Destination}: {{Destination: "JFK", Price: 100.0}},
			{Origin: req.Origin}:      {},
		},
		nil,
	)

	s := New(context.Background(), fake)
	_, err := s.Search(req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify Search was called: at least once for base price + once per explore destination
	if fake.SearchCallCount() < 2 {
		t.Errorf("expected at least 2 Search calls, got %d", fake.SearchCallCount())
	}

	// Verify Explore was called exactly twice (once for origin, once for destination)
	if fake.ExploreCallCount() != 2 {
		t.Errorf("expected 2 Explore calls, got %d", fake.ExploreCallCount())
	}

	// Verify SortByPrice was called at least once (for base price)
	if fake.SortByPriceCallCount() < 1 {
		t.Errorf("expected at least 1 SortByPrice call, got %d", fake.SortByPriceCallCount())
	}
}
