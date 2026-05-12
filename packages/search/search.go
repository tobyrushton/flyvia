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

const maxCalendarCandidates = 5
const defaultResultsBuffer = 128

type Search struct {
	ctx context.Context
	p   provider.Provider

	resultsMu sync.Mutex
	resultsCh chan Result
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

func (s *Search) RegisterResultsChannel(buffer int) <-chan Result {
	if buffer <= 0 {
		buffer = defaultResultsBuffer
	}

	s.resultsMu.Lock()
	defer s.resultsMu.Unlock()
	if s.resultsCh == nil {
		s.resultsCh = make(chan Result, buffer)
	}
	return s.resultsCh
}

func (s *Search) doSearch(req provider.Request) (map[string][]Result, error) {
	defer s.closeResultsChannel()
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

	firstOr, secondOr, err := s.expandOneStopLegs(req, exploreOr, req.Origin, req.Destination, basePrice)
	if err != nil {
		return nil, err
	}
	fmt.Println("expanded origin stop legs")

	firstDest, secondDest, err := s.expandOneStopLegs(req, exploreDest, req.Origin, req.Destination, basePrice)
	if err != nil {
		return nil, err
	}
	fmt.Println("expanded destination stop legs")

	return s.combineItineraries(
		firstOr, secondOr,
		firstDest, secondDest,
	)
}

func (s *Search) emitResult(result Result) {
	s.resultsMu.Lock()
	defer s.resultsMu.Unlock()
	if s.resultsCh == nil {
		return
	}

	s.resultsCh <- result
}

func (s *Search) closeResultsChannel() {
	s.resultsMu.Lock()
	defer s.resultsMu.Unlock()
	if s.resultsCh == nil {
		return
	}
	close(s.resultsCh)
	s.resultsCh = nil
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
	basePrice float64,
) ([][]itinery.Itinery, [][]itinery.Itinery, error) {
	// expand first-leg offers, then use the price calendar to search for valid stopover lengths
	// within budget and boundary constraints, before fetching concrete second-leg itineraries.
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

			windowStart, windowEnd := firstLegDateWindow(first)
			if windowStart.IsZero() || windowEnd.IsZero() || windowEnd.Before(windowStart) {
				return
			}

			calendarReq := provider.Request{
				Origin:        stop,
				Destination:   finalDestination,
				DepartureDate: windowStart,
				ReturnDate:    windowEnd,
				Adults:        req.Adults,
				Children:      req.Children,
				Class:         req.Class,
				Currency:      req.Currency,
			}
			grid, err := s.p.GetPriceCalendar(s.ctx, calendarReq)
			if err != nil {
				s.logExpandError("calendar", stop, finalDestination, err)
				return
			}

			secondForStop := make([]itinery.Itinery, 0)
			secondCache := make(map[string]*secondFetch)
			var cacheMu sync.Mutex
			var secondForStopMu sync.Mutex
			for _, firstItin := range first {
				depMin := normalizeDay(firstItin.Outbound.ArrivalTime)
				retMax := normalizeDay(firstItin.Inbound.DepartureTime)
				if depMin.IsZero() || retMax.IsZero() || retMax.Before(depMin) {
					continue
				}

				budgetCap := basePrice - firstItin.Price
				if budgetCap <= 0 {
					continue
				}

				candidates := gridCandidates(
					grid,
					windowStart,
					depMin,
					retMax,
					budgetCap,
					maxCalendarCandidates,
				)
				var candidatesWg sync.WaitGroup
				for _, candidate := range candidates {
					candidatesWg.Add(1)
					go func() {
						defer candidatesWg.Done()
						key := candidate.Depart.Format("2006-01-02") + "|" + candidate.Return.Format("2006-01-02")

						cacheMu.Lock()
						fetch, ok := secondCache[key]
						if ok {
							cacheMu.Unlock()
							fetch.wg.Wait()
							if len(fetch.itins) == 0 {
								return
							}
							secondForStopMu.Lock()
							secondForStop = append(secondForStop, fetch.itins...)
							secondForStopMu.Unlock()
							return
						}
						fetch = &secondFetch{}
						fetch.wg.Add(1)
						secondCache[key] = fetch
						cacheMu.Unlock()

						secondItins, err := s.p.Search(s.ctx, provider.Request{
							Origin:        stop,
							Destination:   finalDestination,
							DepartureDate: candidate.Depart,
							ReturnDate:    candidate.Return,
							Adults:        req.Adults,
							Children:      req.Children,
							Class:         req.Class,
							Currency:      req.Currency,
						})
						if err != nil {
							s.logExpandError("second-legs", stop, finalDestination, err)
							secondItins = []itinery.Itinery{}
						}
						if len(secondItins) != 0 {
							sort.Slice(secondItins, func(i, j int) bool { return secondItins[i].Price < secondItins[j].Price })
						}

						cacheMu.Lock()
						fetch.itins = secondItins
						fetch.wg.Done()
						cacheMu.Unlock()

						if len(secondItins) == 0 {
							return
						}
						secondForStopMu.Lock()
						secondForStop = append(secondForStop, secondItins...)
						secondForStopMu.Unlock()
					}()
				}
				candidatesWg.Wait()
			}
			if len(secondForStop) == 0 {
				return
			}
			secondMu.Lock()
			secondLegs = append(secondLegs, secondForStop)
			secondMu.Unlock()
		}(stop)
	}
	wg.Wait()

	return firstLegs, secondLegs, nil
}

type gridCandidate struct {
	Depart time.Time
	Return time.Time
	Price  float64
}

type secondFetch struct {
	wg    sync.WaitGroup
	itins []itinery.Itinery
}

func normalizeDay(t time.Time) time.Time {
	if t.IsZero() {
		return time.Time{}
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func firstLegDateWindow(itins []itinery.Itinery) (time.Time, time.Time) {
	var minArrival time.Time
	var maxInbound time.Time
	for _, itin := range itins {
		arr := normalizeDay(itin.Outbound.ArrivalTime)
		dep := normalizeDay(itin.Inbound.DepartureTime)
		if arr.IsZero() || dep.IsZero() {
			continue
		}
		if minArrival.IsZero() || arr.Before(minArrival) {
			minArrival = arr
		}
		if maxInbound.IsZero() || dep.After(maxInbound) {
			maxInbound = dep
		}
	}
	return minArrival, maxInbound
}

func gridCandidates(
	grid [][]float64,
	gridStart time.Time,
	depMin time.Time,
	retMax time.Time,
	budgetCap float64,
	limit int,
) []gridCandidate {
	if len(grid) == 0 || limit <= 0 {
		return nil
	}
	gridSize := len(grid)
	depStartIdx := clampDayIndex(depMin, gridStart, gridSize)
	depEndIdx := clampDayIndex(retMax, gridStart, gridSize)
	retEndIdx := depEndIdx

	candidates := make([]gridCandidate, 0)
	for depIdx := depStartIdx; depIdx <= depEndIdx; depIdx++ {
		row := grid[depIdx]
		for retIdx := depIdx; retIdx <= retEndIdx; retIdx++ {
			if retIdx >= len(row) {
				continue
			}
			price := row[retIdx]
			if price <= 0 || price > budgetCap {
				continue
			}
			candidates = append(candidates, gridCandidate{
				Depart: gridStart.AddDate(0, 0, depIdx),
				Return: gridStart.AddDate(0, 0, retIdx),
				Price:  price,
			})
		}
	}

	sort.Slice(candidates, func(i, j int) bool { return candidates[i].Price < candidates[j].Price })
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	return candidates
}

func clampDayIndex(t time.Time, start time.Time, size int) int {
	if size <= 0 {
		return 0
	}
	if t.Before(start) {
		return 0
	}
	idx := int(t.Sub(start).Hours() / 24)
	if idx < 0 {
		return 0
	}
	if idx >= size {
		return size - 1
	}
	return idx
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
		res := combine.OneStopWithinBounds(stop[0], stop[1])
		for _, r := range res {
			result := NewResult(r.First, r.Second)
			s.emitResult(result)
			results[result.StopCity] = append(results[result.StopCity], result)
		}
	}
	for _, stop := range destStop {
		res := combine.OneStopWithinBounds(stop[0], stop[1])
		for _, r := range res {
			result := NewResult(r.First, r.Second)
			s.emitResult(result)
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
