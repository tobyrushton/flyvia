package search

import (
	"time"

	"github.com/tobyrushton/flyvia/packages/search/itinery"
)

type Result struct {
	StopCity    string
	StopLengths []time.Duration
	Itineries   []itinery.Itinery
	Price       float64
}

func NewResult(
	itinery1, itinery2 itinery.Itinery,
) Result {
	return Result{
		StopCity:  itinery1.Outbound.ArrivalAirport,
		Itineries: []itinery.Itinery{itinery1, itinery2},
		StopLengths: []time.Duration{
			itinery2.Outbound.DepartureTime.Sub(itinery1.Outbound.ArrivalTime),
			itinery1.Inbound.DepartureTime.Sub(itinery2.Inbound.ArrivalTime),
		},
		Price: itinery1.Price + itinery2.Price,
	}
}
