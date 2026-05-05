package leg

import (
	"testing"
	"time"
)

var baseTime = time.Date(2025, 6, 1, 8, 0, 0, 0, time.UTC)

func TestLeg_StructFields(t *testing.T) {
	depTime := baseTime
	arrTime := baseTime.Add(5 * time.Hour)

	l := Leg{
		Flights: []Flight{{
			DepartureTime:    depTime,
			ArrivalTime:      arrTime,
			DepartureAirport: "LHR",
			ArrivalAirport:   "JFK",
			FlightCode:       "BA117",
			Plane:            "Boeing 777",
			Airline:          "British Airways",
		}},
		Stops:            0,
		DepartureTime:    depTime,
		ArrivalTime:      arrTime,
		DepartureAirport: "LHR",
		ArrivalAirport:   "JFK",
		Duration:         5 * time.Hour,
	}

	if l.Stops != 0 {
		t.Errorf("expected 0 stops, got %d", l.Stops)
	}
	if l.ArrivalAirport != "JFK" {
		t.Errorf("expected arrival airport JFK, got %s", l.ArrivalAirport)
	}
	if l.DepartureAirport != "LHR" {
		t.Errorf("expected departure airport LHR, got %s", l.DepartureAirport)
	}
	if l.Duration != 5*time.Hour {
		t.Errorf("expected 5h duration, got %v", l.Duration)
	}
	if len(l.Flights) != 1 {
		t.Fatalf("expected 1 flight, got %d", len(l.Flights))
	}
}

func TestLeg_MultipleFlights(t *testing.T) {
	l := Leg{
		Flights: []Flight{
			{
				DepartureAirport: "LHR",
				ArrivalAirport:   "FRA",
				FlightCode:       "LH901",
			},
			{
				DepartureAirport: "FRA",
				ArrivalAirport:   "JFK",
				FlightCode:       "LH400",
			},
		},
		Stops:            1,
		DepartureAirport: "LHR",
		ArrivalAirport:   "JFK",
	}

	if l.Stops != 1 {
		t.Errorf("expected 1 stop, got %d", l.Stops)
	}
	if len(l.Flights) != 2 {
		t.Fatalf("expected 2 flights, got %d", len(l.Flights))
	}
	if l.Flights[0].ArrivalAirport != "FRA" {
		t.Errorf("expected first flight arrival FRA, got %s", l.Flights[0].ArrivalAirport)
	}
}

func TestLeg_ZeroValue(t *testing.T) {
	var l Leg

	if l.Stops != 0 {
		t.Errorf("expected 0 stops, got %d", l.Stops)
	}
	if l.Duration != 0 {
		t.Errorf("expected 0 duration, got %v", l.Duration)
	}
	if l.Flights != nil {
		t.Errorf("expected nil flights, got %v", l.Flights)
	}
}

func TestFlight_StructFields(t *testing.T) {
	depTime := baseTime
	arrTime := baseTime.Add(2 * time.Hour)

	f := Flight{
		DepartureTime:    depTime,
		ArrivalTime:      arrTime,
		DepartureAirport: "LHR",
		ArrivalAirport:   "CDG",
		FlightCode:       "AF1681",
		Plane:            "Airbus A320",
		Airline:          "Air France",
	}

	if f.DepartureAirport != "LHR" {
		t.Errorf("expected departure LHR, got %s", f.DepartureAirport)
	}
	if f.ArrivalAirport != "CDG" {
		t.Errorf("expected arrival CDG, got %s", f.ArrivalAirport)
	}
	if f.FlightCode != "AF1681" {
		t.Errorf("expected flight code AF1681, got %s", f.FlightCode)
	}
	if f.Plane != "Airbus A320" {
		t.Errorf("expected plane Airbus A320, got %s", f.Plane)
	}
	if f.Airline != "Air France" {
		t.Errorf("expected airline Air France, got %s", f.Airline)
	}
	if !f.DepartureTime.Equal(depTime) {
		t.Errorf("expected departure time %v, got %v", depTime, f.DepartureTime)
	}
	if !f.ArrivalTime.Equal(arrTime) {
		t.Errorf("expected arrival time %v, got %v", arrTime, f.ArrivalTime)
	}
}

func TestFlight_ZeroValue(t *testing.T) {
	var f Flight

	if f.FlightCode != "" {
		t.Errorf("expected empty flight code, got %s", f.FlightCode)
	}
	if f.Plane != "" {
		t.Errorf("expected empty plane, got %s", f.Plane)
	}
	if f.Airline != "" {
		t.Errorf("expected empty airline, got %s", f.Airline)
	}
}
