package combine

import (
	"time"

	"github.com/tobyrushton/flyvia/packages/search/itinery"
)

// CombinedItinery represents a pair of itineraries that form a valid one-stop connection.
type CombinedItinery struct {
	First  itinery.Itinery
	Second itinery.Itinery
}

func OneStop(
	firstItineries, secondItineries []itinery.Itinery,
	minLayover, maxLayover time.Duration,
) []CombinedItinery {
	// this current setup will only connect airports, however a lot of major cities will have multiple airports
	// likely is the case that the flights will fly from different airports as regional and major long haul flights
	// often fly from different airports. This will need to be taken into account later.
	index := make(map[string][]itinery.Itinery)
	for _, itin := range secondItineries {
		index[itin.Outbound.DepartureAirport] = append(index[itin.Outbound.DepartureAirport], itin)
	}

	results := make([]CombinedItinery, 0)

	for _, first := range firstItineries {
		candidates := index[first.Outbound.ArrivalAirport]

		for _, second := range candidates {
			if validLayover(first, second, minLayover, maxLayover) {
				results = append(results, CombinedItinery{First: first, Second: second})
			}
		}
	}

	return results
}

func ConstructStop(first, second [][]itinery.Itinery) map[string][2][]itinery.Itinery {
	// this is a helper function to construct the stop map for the one stop search.
	stopMap := make(map[string][2][]itinery.Itinery)

	for _, f := range first {
		stopMap[f[0].Outbound.ArrivalAirport] = [2][]itinery.Itinery{
			f, nil,
		}
	}

	for _, s := range second {
		if existing, ok := stopMap[s[0].Outbound.DepartureAirport]; ok {
			existing[1] = s
			stopMap[s[0].Outbound.DepartureAirport] = existing
		}
	}

	for key, value := range stopMap {
		if value[0] == nil || value[1] == nil {
			delete(stopMap, key)
		}
	}

	return stopMap
}
