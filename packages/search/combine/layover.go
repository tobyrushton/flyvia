package combine

import (
	"time"

	"github.com/tobyrushton/flyvia/packages/search/itinery"
)

func validLayover(
	first, second itinery.Itinery,
	minLayover, maxLayover time.Duration,
) bool {
	layover := second.Outbound.DepartureTime.Sub(first.Outbound.ArrivalTime)
	return layover >= minLayover && layover <= maxLayover
}

func validBounds(first, second itinery.Itinery) bool {
	if second.Outbound.DepartureTime.Before(first.Outbound.ArrivalTime) {
		return false
	}
	if second.Inbound.ArrivalTime.After(first.Inbound.DepartureTime) {
		return false
	}
	return true
}
