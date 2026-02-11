package itinery

import (
	"github.com/tobyrushton/flyvia/packages/search/leg"
)

type Itinery struct {
	Outbound   leg.Leg
	Inbound    leg.Leg
	Price      float64
	BookingURL string
}

// simplified, wont contain flight details just the price and airports.
type ExploreItinery struct {
	Destination string
	Price       float64
}
