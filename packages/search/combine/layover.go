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
