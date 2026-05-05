package search

import (
	"testing"
	"time"

	"github.com/tobyrushton/flyvia/packages/search/itinery"
	"github.com/tobyrushton/flyvia/packages/search/leg"
)

func TestNewResult_PriceCalculation(t *testing.T) {
	itin1 := itinery.Itinery{
		Outbound: leg.Leg{
			ArrivalAirport: "JFK",
			ArrivalTime:    baseTime.Add(8 * time.Hour),
		},
		Inbound: leg.Leg{
			DepartureTime: baseTime.Add(7*24*time.Hour + 3*time.Hour),
		},
		Price: 300.0,
	}
	itin2 := itinery.Itinery{
		Outbound: leg.Leg{
			DepartureTime: baseTime.Add(12 * time.Hour),
		},
		Inbound: leg.Leg{
			ArrivalTime: baseTime.Add(7*24*time.Hour + 1*time.Hour),
		},
		Price: 200.0,
	}

	r := NewResult(itin1, itin2)

	if r.Price != 500.0 {
		t.Errorf("expected price 500.0, got %f", r.Price)
	}
}

func TestNewResult_StopCity(t *testing.T) {
	itin1 := itinery.Itinery{
		Outbound: leg.Leg{
			ArrivalAirport: "CDG",
			ArrivalTime:    baseTime.Add(2 * time.Hour),
		},
		Inbound: leg.Leg{
			DepartureTime: baseTime.Add(7*24*time.Hour + 5*time.Hour),
		},
		Price: 100.0,
	}
	itin2 := itinery.Itinery{
		Outbound: leg.Leg{
			DepartureTime: baseTime.Add(6 * time.Hour),
		},
		Inbound: leg.Leg{
			ArrivalTime: baseTime.Add(7*24*time.Hour + 2*time.Hour),
		},
		Price: 150.0,
	}

	r := NewResult(itin1, itin2)

	if r.StopCity != "CDG" {
		t.Errorf("expected stop city CDG, got %s", r.StopCity)
	}
}

func TestNewResult_StopLengths(t *testing.T) {
	outboundArrival := baseTime.Add(8 * time.Hour)
	secondDeparture := baseTime.Add(12 * time.Hour)
	// outbound stop = secondDeparture - outboundArrival = 4h

	returnArrival := baseTime.Add(7*24*time.Hour + 1*time.Hour)
	returnDeparture := baseTime.Add(7*24*time.Hour + 4*time.Hour)
	// inbound stop = returnDeparture (itin1.Inbound.DepartureTime) - returnArrival (itin2.Inbound.ArrivalTime) = 3h

	itin1 := itinery.Itinery{
		Outbound: leg.Leg{
			ArrivalAirport: "JFK",
			ArrivalTime:    outboundArrival,
		},
		Inbound: leg.Leg{
			DepartureTime: returnDeparture,
		},
		Price: 300.0,
	}
	itin2 := itinery.Itinery{
		Outbound: leg.Leg{
			DepartureTime: secondDeparture,
		},
		Inbound: leg.Leg{
			ArrivalTime: returnArrival,
		},
		Price: 200.0,
	}

	r := NewResult(itin1, itin2)

	if len(r.StopLengths) != 2 {
		t.Fatalf("expected 2 stop lengths, got %d", len(r.StopLengths))
	}

	expectedOutbound := 4 * time.Hour
	if r.StopLengths[0] != expectedOutbound {
		t.Errorf("expected outbound stop length %v, got %v", expectedOutbound, r.StopLengths[0])
	}

	expectedInbound := 3 * time.Hour
	if r.StopLengths[1] != expectedInbound {
		t.Errorf("expected inbound stop length %v, got %v", expectedInbound, r.StopLengths[1])
	}
}

func TestNewResult_Itineraries(t *testing.T) {
	itin1 := itinery.Itinery{
		Outbound: leg.Leg{
			ArrivalAirport:   "ORD",
			DepartureAirport: "LHR",
			ArrivalTime:      baseTime.Add(10 * time.Hour),
		},
		Inbound: leg.Leg{
			DepartureTime: baseTime.Add(7 * 24 * time.Hour),
		},
		Price:      400.0,
		BookingURL: "https://example.com/1",
	}
	itin2 := itinery.Itinery{
		Outbound: leg.Leg{
			DepartureAirport: "ORD",
			ArrivalAirport:   "LAX",
			DepartureTime:    baseTime.Add(14 * time.Hour),
		},
		Inbound: leg.Leg{
			ArrivalTime: baseTime.Add(7*24*time.Hour - 4*time.Hour),
		},
		Price:      250.0,
		BookingURL: "https://example.com/2",
	}

	r := NewResult(itin1, itin2)

	if len(r.Itineries) != 2 {
		t.Fatalf("expected 2 itineraries, got %d", len(r.Itineries))
	}
	if r.Itineries[0].BookingURL != "https://example.com/1" {
		t.Error("first itinerary not preserved correctly")
	}
	if r.Itineries[1].BookingURL != "https://example.com/2" {
		t.Error("second itinerary not preserved correctly")
	}
}

func TestNewResult_ZeroPrices(t *testing.T) {
	itin1 := itinery.Itinery{
		Outbound: leg.Leg{ArrivalAirport: "X", ArrivalTime: baseTime},
		Inbound:  leg.Leg{DepartureTime: baseTime.Add(24 * time.Hour)},
		Price:    0.0,
	}
	itin2 := itinery.Itinery{
		Outbound: leg.Leg{DepartureTime: baseTime.Add(4 * time.Hour)},
		Inbound:  leg.Leg{ArrivalTime: baseTime.Add(20 * time.Hour)},
		Price:    0.0,
	}

	r := NewResult(itin1, itin2)
	if r.Price != 0.0 {
		t.Errorf("expected price 0, got %f", r.Price)
	}
}

func TestNewResult_NegativeStopLength(t *testing.T) {
	// If second departs before first arrives, stop length is negative
	itin1 := itinery.Itinery{
		Outbound: leg.Leg{
			ArrivalAirport: "JFK",
			ArrivalTime:    baseTime.Add(12 * time.Hour),
		},
		Inbound: leg.Leg{
			DepartureTime: baseTime.Add(7*24*time.Hour + 1*time.Hour),
		},
		Price: 300.0,
	}
	itin2 := itinery.Itinery{
		Outbound: leg.Leg{
			DepartureTime: baseTime.Add(8 * time.Hour), // before first arrives
		},
		Inbound: leg.Leg{
			ArrivalTime: baseTime.Add(7*24*time.Hour + 5*time.Hour), // after first departs
		},
		Price: 200.0,
	}

	r := NewResult(itin1, itin2)

	// Stop length should be negative
	if r.StopLengths[0] >= 0 {
		t.Errorf("expected negative outbound stop length, got %v", r.StopLengths[0])
	}
}
