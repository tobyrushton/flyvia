package averages

import (
	"time"

	"github.com/tobyrushton/flyvia/packages/search/itinery"
)

// avg duration, stops, price
func Calculate(itins []itinery.Itinery) (time.Duration, float64, float64) {
	var totalDuration time.Duration
	var totalStops int
	var totalPrice float64

	for _, itin := range itins {
		totalDuration += itin.Outbound.ArrivalTime.Sub(itin.Outbound.DepartureTime) + itin.Inbound.ArrivalTime.Sub(itin.Inbound.DepartureTime)
		totalStops += (itin.Outbound.Stops + itin.Inbound.Stops) / 2
		totalPrice += itin.Price
	}

	count := float64(len(itins))
	avgDuration := time.Duration(float64(totalDuration) / count)
	avgStops := float64(totalStops) / count
	avgPrice := totalPrice / count

	return avgDuration, avgStops, avgPrice
}
