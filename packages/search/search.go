package search

import (
	"context"
	"sort"
	"sync"

	"github.com/tobyrushton/flyvia/packages/search/averages"
	"github.com/tobyrushton/flyvia/packages/search/itinery"
	"github.com/tobyrushton/flyvia/packages/search/provider"
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
	return nil, nil
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
