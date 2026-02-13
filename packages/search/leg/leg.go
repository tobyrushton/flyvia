package leg

import "time"

type Leg struct {
	Flights          []Flight
	Stops            int
	DepartureTime    time.Time
	ArrivalTime      time.Time
	DepartureAirport string
	ArrivalAirport   string
	Duration         time.Duration
}

type Flight struct {
	DepartureTime    time.Time
	ArrivalTime      time.Time
	DepartureAirport string
	ArrivalAirport   string
	FlightCode       string
	Plane            string
	Airline          string
}
