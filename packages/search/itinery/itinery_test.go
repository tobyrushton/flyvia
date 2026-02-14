package itinery

import (
	"testing"

	"github.com/tobyrushton/flyvia/packages/search/leg"
)

func TestItinery_StructFields(t *testing.T) {
	outbound := leg.Leg{
		DepartureAirport: "LHR",
		ArrivalAirport:   "JFK",
	}
	inbound := leg.Leg{
		DepartureAirport: "JFK",
		ArrivalAirport:   "LHR",
	}

	it := Itinery{
		Outbound:   outbound,
		Inbound:    inbound,
		Price:      500.0,
		BookingURL: "https://example.com",
	}

	if it.Outbound.DepartureAirport != "LHR" {
		t.Errorf("expected outbound departure LHR, got %s", it.Outbound.DepartureAirport)
	}
	if it.Outbound.ArrivalAirport != "JFK" {
		t.Errorf("expected outbound arrival JFK, got %s", it.Outbound.ArrivalAirport)
	}
	if it.Inbound.DepartureAirport != "JFK" {
		t.Errorf("expected inbound departure JFK, got %s", it.Inbound.DepartureAirport)
	}
	if it.Inbound.ArrivalAirport != "LHR" {
		t.Errorf("expected inbound arrival LHR, got %s", it.Inbound.ArrivalAirport)
	}
	if it.Price != 500.0 {
		t.Errorf("expected price 500.0, got %f", it.Price)
	}
	if it.BookingURL != "https://example.com" {
		t.Errorf("expected booking URL https://example.com, got %s", it.BookingURL)
	}
}

func TestItinery_ZeroValue(t *testing.T) {
	var it Itinery

	if it.Price != 0 {
		t.Errorf("expected zero price, got %f", it.Price)
	}
	if it.BookingURL != "" {
		t.Errorf("expected empty booking URL, got %s", it.BookingURL)
	}
}

func TestExploreItinery_StructFields(t *testing.T) {
	ei := ExploreItinery{
		Destination: "CDG",
		Price:       150.0,
	}

	if ei.Destination != "CDG" {
		t.Errorf("expected destination CDG, got %s", ei.Destination)
	}
	if ei.Price != 150.0 {
		t.Errorf("expected price 150.0, got %f", ei.Price)
	}
}

func TestExploreItinery_ZeroValue(t *testing.T) {
	var ei ExploreItinery

	if ei.Destination != "" {
		t.Errorf("expected empty destination, got %s", ei.Destination)
	}
	if ei.Price != 0 {
		t.Errorf("expected zero price, got %f", ei.Price)
	}
}
