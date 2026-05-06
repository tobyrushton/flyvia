package search

import (
	"context"
	"fmt"
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

func (s *Search) Search(req provider.Request) (map[string][]Result, error) {
	return s.doSearch(req)
}

func (s *Search) doSearch(req provider.Request) (map[string][]Result, error) {
	// explore origins and destinations in parallel
	// expand reasonable first legs to get actual itineries with flight prices.
	// then expand these to get the second legs of the journeys.
	// then we need to combine these into valid one stop journeys.
	// group by stopover city and sort each group by price.
	basePrice, err := s.getBasePrice(req)
	if err != nil {
		return nil, err
	}
	fmt.Println("found base price")
	fmt.Printf("got: %f\n", basePrice)
	exploreOr, exploreDest, err := s.explore(req)
	if err != nil {
		return nil, err
	}
	fmt.Println("found explore dests")
	exploreOr = s.filterReasonableItineraries(exploreOr, basePrice)
	exploreDest = s.filterReasonableItineraries(exploreDest, basePrice)

	firstOr, secondOr, err := s.expandOneStopLegs(req, exploreOr, req.Origin, req.Destination)
	if err != nil {
		return nil, err
	}
	fmt.Println("expanded origin stop legs")

	firstDest, secondDest, err := s.expandOneStopLegs(req, exploreDest, req.Origin, req.Destination)
	if err != nil {
		return nil, err
	}
	fmt.Println("expanded destination stop legs")

	return s.combineItineraries(
		firstOr, secondOr,
		firstDest, secondDest,
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
	// fmt.Printf("cheapest price: %+v\n", offers[0])

	durationScore := avgDuration.Hours() / (avgDuration.Hours() + 6.0)
	stopsScore := avgStops / (avgStops + 1.0)
	qualityAdjustment := 1.0 + 0.3*durationScore + 0.2*stopsScore

	blendedPrice := 0.7*cheapestPrice + 0.3*avgPrice
	basePrice := blendedPrice * qualityAdjustment

	return basePrice, nil
}

func (s *Search) expandOneStopLegs(
	req provider.Request,
	exploreItineries []itinery.ExploreItinery,
	firstOrigin string,
	finalDestination string,
) ([][]itinery.Itinery, [][]itinery.Itinery, error) {
	// expand first-leg offers, then expand second-leg offers using the earliest arrival as the outbound time.
	wg := sync.WaitGroup{}
	firstLegs := make([][]itinery.Itinery, 0)
	secondLegs := make([][]itinery.Itinery, 0)
	var firstMu sync.Mutex
	var secondMu sync.Mutex

	for _, exploreItinery := range exploreItineries {
		stop := exploreItinery.Destination
		wg.Add(1)
		go func(stop string) {
			defer wg.Done()
			first, err := s.p.Search(s.ctx, provider.Request{
				Origin:        firstOrigin,
				Destination:   stop,
				DepartureDate: req.DepartureDate,
				ReturnDate:    req.ReturnDate,
				Adults:        req.Adults,
				Children:      req.Children,
				Class:         req.Class,
				Currency:      req.Currency,
			})
			if err != nil {
				s.logExpandError("first-legs", firstOrigin, stop, err)
				return
			}
			if len(first) == 0 {
				return
			}
			sort.Slice(first, func(i, j int) bool { return first[i].Price < first[j].Price })
			firstMu.Lock()
			firstLegs = append(firstLegs, first)
			firstMu.Unlock()

			outboundTime := earliestArrival(first)
			if outboundTime.IsZero() {
				outboundTime = req.DepartureDate
			}
			second, err := s.p.Search(s.ctx, provider.Request{
				Origin:        stop,
				Destination:   finalDestination,
				DepartureDate: outboundTime,
				ReturnDate:    req.ReturnDate,
				Adults:        req.Adults,
				Children:      req.Children,
				Class:         req.Class,
				Currency:      req.Currency,
			})
			if err != nil {
				s.logExpandError("second-legs", stop, finalDestination, err)
				return
			}
			if len(second) == 0 {
				return
			}
			sort.Slice(second, func(i, j int) bool { return second[i].Price < second[j].Price })
			secondMu.Lock()
			secondLegs = append(secondLegs, second)
			secondMu.Unlock()
		}(stop)
	}
	wg.Wait()

	return firstLegs, secondLegs, nil
}

func earliestArrival(itineries []itinery.Itinery) time.Time {
	var earliest time.Time
	for _, itin := range itineries {
		if itin.Outbound.ArrivalTime.IsZero() {
			continue
		}
		if earliest.IsZero() || itin.Outbound.ArrivalTime.Before(earliest) {
			earliest = itin.Outbound.ArrivalTime
		}
	}
	return earliest
}

func (s *Search) logExpandError(phase, origin, destination string, err error) {
	if err == nil {
		return
	}
	fmt.Printf("expand %s failed for %s -> %s: %v\n", phase, origin, destination, err)
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
) (map[string][]Result, error) {
	// we want to combine the first and second legs of the itineraries to get valid one stop journeys.
	// we can do this by iterating over the first legs and then finding the matching second legs.
	// we can then calculate the total price and duration of the journey and sort by price.

	orStop := combine.ConstructStop(firstOr, secondOr)
	destStop := combine.ConstructStop(firstDest, secondDest)

	results := make(map[string][]Result)
	for _, stop := range orStop {
		res := combine.OneStop(stop[0], stop[1], minLayover, maxLayover)
		for _, r := range res {
			result := NewResult(r.First, r.Second)
			results[result.StopCity] = append(results[result.StopCity], result)
		}
	}
	for _, stop := range destStop {
		res := combine.OneStop(stop[0], stop[1], minLayover, maxLayover)
		for _, r := range res {
			result := NewResult(r.First, r.Second)
			results[result.StopCity] = append(results[result.StopCity], result)
		}
	}

	for stopCity := range results {
		sort.Slice(results[stopCity], func(i, j int) bool {
			return results[stopCity][i].Price < results[stopCity][j].Price
		})
	}
	return results, nil
}
