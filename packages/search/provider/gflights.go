package provider

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/tobyrushton/flyvia/packages/search/itinery"
	"github.com/tobyrushton/flyvia/packages/search/leg"
	"github.com/tobyrushton/gflights"
	"github.com/tobyrushton/gflights/iata"
)

type GFlights struct {
	s *gflights.Session
}

func NewGFlights(proxy string) (*GFlights, error) {
	client, err := NewBrowserClient(BrowserClientOptions{
		ProxyURL: proxy,
	})
	if err != nil {
		return nil, err
	}

	s, err := gflights.New(gflights.WithClient(client))
	if err != nil {
		return nil, err
	}

	return &GFlights{
		s: s,
	}, nil
}

func (g *GFlights) Explore(
	ctx context.Context,
	req Request,
	origin string,
) ([]itinery.ExploreItinery, error) {
	srcCities, srcAirports := g.sortLocations([]string{origin})

	offers, err := g.s.GetExplore(ctx, gflights.ExploreArgs{
		DepartureDate: req.DepartureDate,
		ReturnDate:    req.ReturnDate,
		SrcCities:     srcCities,
		SrcAirports:   srcAirports,
		Options: gflights.Options{
			Travelers: gflights.Travelers{
				Adults:   req.Adults,
				Children: req.Children,
			},
			Class:    gflights.Class(req.Class),
			Currency: req.Currency,
			TripType: gflights.RoundTrip,
			Stops:    gflights.AnyStops,
		},
	})

	if err != nil {
		return nil, err
	}

	ei := make([]itinery.ExploreItinery, 0)

	for _, offer := range offers {
		ei = append(ei, itinery.ExploreItinery{
			Destination: offer.AirportCode,
			Price:       float64(offer.Price),
		})
	}

	return ei, nil
}

func (g *GFlights) Search(
	ctx context.Context,
	req Request,
) ([]itinery.Itinery, error) {
	srcCities, srcAirports := g.sortLocations([]string{req.Origin})
	dstCities, dstAirports := g.sortLocations([]string{req.Destination})

	outboundFlights, _, err := g.s.GetOutboundOffers(ctx, gflights.Args{
		DepartureDate: req.DepartureDate,
		ReturnDate:    req.ReturnDate,
		SrcCities:     srcCities,
		SrcAirports:   srcAirports,
		DstCities:     dstCities,
		DstAirports:   dstAirports,
		Options: gflights.Options{
			Travelers: gflights.Travelers{
				Adults:   req.Adults,
				Children: req.Children,
			},
			Class:    gflights.Class(req.Class),
			Currency: req.Currency,
			TripType: gflights.RoundTrip,
			Stops:    gflights.AnyStops,
		},
	})
	if err != nil {
		return nil, err
	}
	if len(outboundFlights) == 0 {
		return nil, nil
	}

	// sort outboundFlights and lets choose top x
	sort.Slice(outboundFlights, func(i, j int) bool {
		return outboundFlights[i].Price < outboundFlights[j].Price
	})

	itineries := make([]itinery.Itinery, 0)
	wg := sync.WaitGroup{}
	legsMu := sync.Mutex{}

	capPrice := outboundFlights[0].Price * 1.5

	for i := 0; i < 5 && i < len(outboundFlights); i++ {
		wg.Add(1)
		go func(of gflights.OutboundOffer) {
			defer wg.Done()

			returnFlights, err := of.GetReturnFlights(ctx)
			if err != nil {
				return
			}

			for _, rf := range returnFlights {
				if rf.Price > 0 && rf.Price <= capPrice {
					t, err := of.SelectReturnFlight(rf)
					if err != nil {
						fmt.Println("Error selecting return flight:", err)
						continue
					}
					url, err := g.s.SerialiseBookingURL(ctx, t)
					if err != nil {
						fmt.Println("Error serialising booking URL:", err)
						continue
					}

					if len(of.Flight) == 0 {
						fmt.Println("Error: outbound offer has no flight legs")
						continue
					}
					outboundDep := of.Flight[0].DepTime
					outboundArr := of.Flight[len(of.Flight)-1].ArrTime

					legsMu.Lock()
					itineries = append(itineries, itinery.Itinery{
						Outbound: leg.Leg{
							DepartureAirport: of.SrcAirportCode,
							ArrivalAirport:   of.DstAirportCode,
							DepartureTime:    outboundDep,
							ArrivalTime:      outboundArr,
							Stops:            len(of.Flight) - 1,
							Flights:          gflightsFlightsToLegFlights(of.Flight),
							Duration:         outboundArr.Sub(outboundDep),
						},
						Inbound: leg.Leg{
							DepartureAirport: rf.Flight[0].DepAirportCode,
							ArrivalAirport:   rf.Flight[len(rf.Flight)-1].ArrAirportCode,
							DepartureTime:    rf.Flight[0].DepTime,
							ArrivalTime:      rf.Flight[len(rf.Flight)-1].ArrTime,
							Stops:            len(rf.Flight) - 1,
							Flights:          gflightsFlightsToLegFlights(rf.Flight),
							Duration:         rf.Flight[len(rf.Flight)-1].ArrTime.Sub(rf.Flight[0].DepTime),
						},
						Price:      rf.Price,
						BookingURL: url,
					})
					legsMu.Unlock()
				}
			}

		}(outboundFlights[i])
	}

	wg.Wait()

	return itineries, nil
}

func (g *GFlights) SortByPrice(itins *[]itinery.Itinery) {
	sort.Slice(*itins, func(i, j int) bool {
		return (*itins)[i].Price < (*itins)[j].Price
	})
}

func (g *GFlights) GetPriceCalendar(
	ctx context.Context,
	req Request,
) ([][]float64, error) {
	normalizeDay := func(t time.Time) time.Time {
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	}

	srcCities, srcAirports := g.sortLocations([]string{req.Origin})
	dstCities, dstAirports := g.sortLocations([]string{req.Destination})

	startDepart := normalizeDay(req.DepartureDate)
	endDepart := normalizeDay(req.ReturnDate)
	startReturn := normalizeDay(req.DepartureDate)
	endReturn := normalizeDay(req.ReturnDate)

	offers, err := g.s.GetPriceGrid(
		ctx,
		gflights.PriceGridArgs{
			StartDepartureRange: startDepart,
			EndDepartureRange:   endDepart,
			StartReturnRange:    startReturn,
			EndReturnRange:      endReturn,
			SrcCities:           srcCities,
			SrcAirports:         srcAirports,
			DstCities:           dstCities,
			DstAirports:         dstAirports,
			Options: gflights.Options{
				Travelers: gflights.Travelers{
					Adults:   req.Adults,
					Children: req.Children,
				},
				Class:    gflights.Class(req.Class),
				Currency: req.Currency,
				TripType: gflights.RoundTrip,
				Stops:    gflights.AnyStops,
			},
		},
	)
	if err != nil {
		return nil, err
	}

	depDays := int(endDepart.Sub(startDepart).Hours()/24) + 1
	retDays := int(endReturn.Sub(startReturn).Hours()/24) + 1
	if depDays <= 0 || retDays <= 0 {
		return [][]float64{}, nil
	}

	grid := make([][]float64, depDays)
	for i := range grid {
		grid[i] = make([]float64, retDays)
	}

	for _, offer := range offers {
		if offer.Price <= 0 {
			continue
		}
		dep := normalizeDay(offer.DepartureDate)
		ret := normalizeDay(offer.ReturnDate)
		row := int(dep.Sub(startDepart).Hours() / 24)
		col := int(ret.Sub(startReturn).Hours() / 24)
		if row < 0 || row >= depDays || col < 0 || col >= retDays {
			continue
		}
		if grid[row][col] == 0 || offer.Price < grid[row][col] {
			grid[row][col] = offer.Price
		}
	}

	return grid, nil
}

func (g *GFlights) sortLocations(locations []string) ([]string, []string) {
	cities := make([]string, 0)
	airports := make([]string, 0)

	for _, loc := range locations {
		tz := iata.IATATimeZone(loc)
		if tz.City == "Not supported IATA Code" {
			cities = append(cities, loc)
		} else {
			airports = append(airports, loc)
		}
	}
	return cities, airports
}

func gflightsFlightToLegFlight(gf gflights.Flight) leg.Flight {
	return leg.Flight{
		DepartureTime:    gf.DepTime,
		ArrivalTime:      gf.ArrTime,
		DepartureAirport: gf.DepAirportCode,
		ArrivalAirport:   gf.ArrAirportCode,
		FlightCode:       gf.FlightCode.AirlineCode + gf.FlightCode.FlightNumber,
		Plane:            gf.Airplane,
		Airline:          gf.AirlineName,
	}
}

func gflightsFlightsToLegFlights(gfs []gflights.Flight) []leg.Flight {
	flights := make([]leg.Flight, len(gfs))
	for i, gf := range gfs {
		flights[i] = gflightsFlightToLegFlight(gf)
	}
	return flights
}

var _ Provider = (*GFlights)(nil)
