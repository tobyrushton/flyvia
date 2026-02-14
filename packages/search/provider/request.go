package provider

import (
	"time"

	"golang.org/x/text/currency"
)

type Class int64

const (
	Economy Class = iota + 1
	PremiumEconomy
	Business
	First
)

type Request struct {
	Origin      string
	Destination string

	DepartureDate time.Time
	ReturnDate    time.Time

	Adults   int
	Children int

	Class    Class
	Currency currency.Unit
}
