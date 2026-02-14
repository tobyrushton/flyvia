package search

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/tobyrushton/flyvia/packages/search/averages"
	"github.com/tobyrushton/flyvia/packages/search/combine"
	"github.com/tobyrushton/flyvia/packages/search/itinery"
	"github.com/tobyrushton/flyvia/packages/search/provider"
)

const (
	minLayover = 3 * time.Hour
	maxLayover = 6 * time.Hour
)

type Search struct {
	ctx context.Context
	p   provider.Provider
}

func New(
	ctx context.Context,
	provider provider.Provider,
) *Search {
	return &Search{
		ctx: ctx,
		p:   provider,
	}
}

func (s *Search) Search(req provider.Request) ([]Result, error) {
	return nil, nil
}

func (s *Search) doSearch(req provider.Request) ([]Result, error) {
	// explore origins and destinations in parallel
	// expand reasonable first legs to get actual itineries with flight prices.
	// then expand these to get the second legs of the journeys.
	// then we need to combine these into valid one stop journeys.
	// sort by price and return.
	basePrice, err := s.getBasePrice(req)
	if err != nil {
		return nil, err
	}

	exploreOr, exploreDest, err := s.explore(req)
	if err != nil {
		return nil, err
	}

	exploreOr = s.filterReasonableItineraries(exploreOr, basePrice)
	exploreDest = s.filterReasonableItineraries(exploreDest, basePrice)

	itineriesOrigin, err := s.expandFirstLegs(req, exploreOr)
	if err != nil {
		return nil, err
	}

	itineriesDest, err := s.expandFirstLegs(req, exploreDest)
	if err != nil {
		return nil, err
	}

	secondOr, secondDest, err := s.expandSecondLegs(req, exploreOr, exploreDest)
	if err != nil {
		return nil, err
	}

	return s.combineItineraries(
		itineriesOrigin, secondOr,
		itineriesDest, secondDest,
	)
}

func (s *Search) explore(req provider.Request) ([]itinery.ExploreItinery, []itinery.ExploreItinery, error) {
	origins := []string{req.Destination, req.Origin}

	wg := sync.WaitGroup{}
	res := make([][]itinery.ExploreItinery, len(origins))
	var exploreErr error

	for i, origin := range origins {
		wg.Add(1)
		go func(i int, origin string) {
			defer wg.Done()
			it, err := s.p.Explore(s.ctx, req, origin)
			if err != nil {
				exploreErr = err
				return
			}
			sort.Slice(it, func(i, j int) bool { return it[i].Price < it[j].Price })
			res[i] = it
		}(i, origin)
	}
	wg.Wait()

	if exploreErr != nil {
		return nil, nil, exploreErr
	}
	return res[0], res[1], nil
}

// here we want to establish what is a reasonable base price for the journey if it was bought as one ticket
// some considerations need to be made here like taking a route that normally has a 10hr stopover should be
// considered not a reasonable route.
func (s *Search) getBasePrice(req provider.Request) (float64, error) {
	offers, err := s.p.Search(s.ctx, req)
	if err != nil {
		return 0, err
	}
	if len(offers) == 0 {
		return 0, nil
	}

	avgDuration, avgStops, avgPrice := averages.Calculate(offers)
	s.p.SortByPrice(&offers)

	// Calculate a reasonable base price using avgPrice, avgDuration, and avgStops.
	//
	// The base price represents what a user should reasonably expect to pay for a single
	// ticket on this route. Split-ticket deals must beat this threshold to be worthwhile.
	//
	// Logic:
	// 1. Blend the cheapest and average price (70/30) as the anchor.
	// 2. Apply a quality adjustment based on duration and stops:
	//    - Poor quality direct routes (long duration, many stops) push the threshold UP,
	//      making it easier for split tickets to beat — users are more willing to try
	//      alternatives when the direct option is bad.
	//    - High quality direct routes (short, few stops) keep the threshold LOW,
	//      meaning split tickets need a significant saving to justify the hassle.
	//
	// durationScore: 0 → instant, 0.5 → 6hrs, 0.67 → 12hrs, 0.8 → 24hrs
	// stopsScore:    0 → direct, 0.5 → 1 stop, 0.67 → 2 stops
	cheapestPrice := offers[0].Price

	durationScore := avgDuration.Hours() / (avgDuration.Hours() + 6.0)
	stopsScore := avgStops / (avgStops + 1.0)
	qualityAdjustment := 1.0 + 0.3*durationScore + 0.2*stopsScore

	blendedPrice := 0.7*cheapestPrice + 0.3*avgPrice
	basePrice := blendedPrice * qualityAdjustment

	return basePrice, nil
}

func (s *Search) expandFirstLegs(req provider.Request, exploreItineries []itinery.ExploreItinery) ([][]itinery.Itinery, error) {
	// we want to expand the first legs of the explore itineries to get actual flight offers with prices.
	// we can do this in parallel and then sort by price.
	wg := sync.WaitGroup{}
	itineries := make([][]itinery.Itinery, len(exploreItineries))
	var expandErr error

	for i, exploreItinery := range exploreItineries {
		wg.Add(1)
		go func(i int, exploreItinery itinery.ExploreItinery) {
			defer wg.Done()
			it, err := s.p.Search(s.ctx, provider.Request{
				Origin:        req.Origin,
				Destination:   exploreItinery.Destination,
				DepartureDate: req.DepartureDate,
				ReturnDate:    req.ReturnDate,
				Adults:        req.Adults,
				Children:      req.Children,
				Class:         req.Class,
				Currency:      req.Currency,
			})
			if err != nil {
				expandErr = err
				return
			}
			sort.Slice(it, func(i, j int) bool { return it[i].Price < it[j].Price })
			itineries[i] = it
		}(i, exploreItinery)
	}
	wg.Wait()

	return itineries, expandErr
}

func (s *Search) expandSecondLegs(
	req provider.Request,
	exploreOr []itinery.ExploreItinery,
	exploreDest []itinery.ExploreItinery,
) ([][]itinery.Itinery, [][]itinery.Itinery, error) {

	// we want to expand the second legs of the explore itineraries to get actual flight offers with prices.
	// we can do this in parallel and then sort by price.

	wg := sync.WaitGroup{}
	itineriesOrigin := make([][]itinery.Itinery, len(exploreOr))
	itineriesDest := make([][]itinery.Itinery, len(exploreDest))
	var expandErr error

	for i, exploreItinery := range exploreOr {
		wg.Add(1)
		go func(i int, exploreItinery itinery.ExploreItinery) {
			defer wg.Done()
			it, err := s.p.Search(s.ctx, provider.Request{
				Origin:        exploreItinery.Destination,
				Destination:   req.Destination,
				DepartureDate: req.DepartureDate,
				ReturnDate:    req.ReturnDate,
				Adults:        req.Adults,
				Children:      req.Children,
				Class:         req.Class,
				Currency:      req.Currency,
			})
			if err != nil {
				expandErr = err
				return
			}
			sort.Slice(it, func(i, j int) bool { return it[i].Price < it[j].Price })
			itineriesOrigin[i] = it
		}(i, exploreItinery)
	}

	for i, exploreItinery := range exploreDest {
		wg.Add(1)
		go func(i int, exploreItinery itinery.ExploreItinery) {
			defer wg.Done()
			it, err := s.p.Search(s.ctx, provider.Request{
				Origin:        req.Origin,
				Destination:   exploreItinery.Destination,
				DepartureDate: req.DepartureDate,
				ReturnDate:    req.ReturnDate,
				Adults:        req.Adults,
				Children:      req.Children,
				Class:         req.Class,
				Currency:      req.Currency,
			})
			if err != nil {
				expandErr = err
				return
			}
			sort.Slice(it, func(i, j int) bool { return it[i].Price < it[j].Price })
			itineriesDest[i] = it
		}(i, exploreItinery)
	}
	wg.Wait()

	return itineriesOrigin, itineriesDest, expandErr
}

func (s *Search) filterReasonableItineraries(
	itineries []itinery.ExploreItinery,
	basePrice float64,
) []itinery.ExploreItinery {
	reasonableItineries := make([]itinery.ExploreItinery, 0)
	for _, itin := range itineries {
		if itin.Price > basePrice*0.8 {
			continue
		}
		reasonableItineries = append(reasonableItineries, itin)
	}
	return reasonableItineries
}

func (s *Search) combineItineraries(
	firstOr, secondOr, firstDest, secondDest [][]itinery.Itinery,
) ([]Result, error) {
	// we want to combine the first and second legs of the itineraries to get valid one stop journeys.
	// we can do this by iterating over the first legs and then finding the matching second legs.
	// we can then calculate the total price and duration of the journey and sort by price.

	orStop := combine.ConstructStop(firstOr, secondOr)
	destStop := combine.ConstructStop(firstDest, secondDest)

	results := []Result{}
	for _, stop := range orStop {
		res := combine.OneStop(stop[0], stop[1], minLayover, maxLayover)
		for _, r := range res {
			results = append(results, NewResult(r.First, r.Second))
		}
	}
	for _, stop := range destStop {
		res := combine.OneStop(stop[0], stop[1], minLayover, maxLayover)
		for _, r := range res {
			results = append(results, NewResult(r.First, r.Second))
		}
	}

	sort.Slice(results, func(i, j int) bool { return results[i].Price < results[j].Price })
	return results, nil
}
