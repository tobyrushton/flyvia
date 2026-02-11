package itinery

import "github.com/tobyrushton/flyvia/packages/search/leg"

type Itinery struct {
	Outbound   leg.Leg
	Inbound    leg.Leg
	Price      float64
	BookingURL string
}
