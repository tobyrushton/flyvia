package provider

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/tobyrushton/flyvia/packages/search/itinery"
	"github.com/tobyrushton/flyvia/packages/search/leg"
	"github.com/tobyrushton/gflights"
)

type GFlights struct {
	s *gflights.Session
}

func NewGFlights() (*GFlights, error) {
	s, err := gflights.New()
	if err != nil {
		return nil, err
	}

	return &GFlights{
		s: s,
	}, nil
}

func (g *GFlights) Explore(
	ctx context.Context,
	origin string,
) ([]itinery.Itinery, error) {
	return nil, nil
}

func (g *GFlights) Search(
	ctx context.Context,
	req Request,
) ([]itinery.Itinery, error) {
	outboundFlights, _, err := g.s.GetOutboundOffers(ctx, gflights.Args{
		DepartureDate: req.DepartureDate,
		ReturnDate:    req.ReturnDate,
		SrcCities:     []string{req.Origin},
		DstCities:     []string{req.Destination},
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

	// sort outboundFlights and lets choose top x
	sort.Slice(outboundFlights, func(i, j int) bool {
		return outboundFlights[i].Price < outboundFlights[j].Price
	})

	itineries := make([]itinery.Itinery, 0)
	wg := sync.WaitGroup{}
	legsMu := sync.Mutex{}

	capPrice := outboundFlights[5].Price

	for i := 0; i < 5 && i < len(outboundFlights); i++ {
		wg.Add(1)
		go func(of gflights.OutboundOffer) {
			defer wg.Done()

			returnFlights, err := of.GetReturnFlights(ctx)
			if err != nil {
				return
			}

			for _, rf := range returnFlights {
				if rf.Price <= capPrice {
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

					legsMu.Lock()
					itineries = append(itineries, itinery.Itinery{
						Outbound: leg.Leg{
							DepartureAirport: of.SrcAirportCode,
							ArrivalAirport:   of.DstAirportCode,
							DepartureTime:    of.DepartureDate,
							ArrivalTime:      of.ReturnDate,
							Stops:            len(of.Flight) - 1,
							Flights:          gflightsFlightsToLegFlights(of.Flight),
						},
						Inbound: leg.Leg{
							DepartureAirport: rf.Flight[0].DepAirportCode,
							ArrivalAirport:   rf.Flight[len(rf.Flight)-1].ArrAirportCode,
							DepartureTime:    rf.Flight[0].DepTime,
							ArrivalTime:      rf.Flight[len(rf.Flight)-1].ArrTime,
							Stops:            len(rf.Flight) - 1,
							Flights:          gflightsFlightsToLegFlights(rf.Flight),
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
