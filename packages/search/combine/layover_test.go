package combine

import (
	"testing"
	"time"

	"github.com/tobyrushton/flyvia/packages/search/itinery"
	"github.com/tobyrushton/flyvia/packages/search/leg"
)

func TestValidLayover_WithinBounds(t *testing.T) {
	baseTime := time.Date(2025, 6, 1, 8, 0, 0, 0, time.UTC)

	first := itinery.Itinery{
		Outbound: leg.Leg{ArrivalTime: baseTime.Add(8 * time.Hour)},
	}
	second := itinery.Itinery{
		Outbound: leg.Leg{DepartureTime: baseTime.Add(12 * time.Hour)}, // 4h layover
	}

	if !validLayover(first, second, 1*time.Hour, 6*time.Hour) {
		t.Error("expected valid layover for 4h between 1h-6h bounds")
	}
}

func TestValidLayover_ExactMinimum(t *testing.T) {
	baseTime := time.Date(2025, 6, 1, 8, 0, 0, 0, time.UTC)

	first := itinery.Itinery{
		Outbound: leg.Leg{ArrivalTime: baseTime.Add(8 * time.Hour)},
	}
	second := itinery.Itinery{
		Outbound: leg.Leg{DepartureTime: baseTime.Add(9 * time.Hour)}, // exactly 1h
	}

	// >= minLayover
	if !validLayover(first, second, 1*time.Hour, 6*time.Hour) {
		t.Error("expected valid layover at exact minimum boundary")
	}
}

func TestValidLayover_ExactMaximum(t *testing.T) {
	baseTime := time.Date(2025, 6, 1, 8, 0, 0, 0, time.UTC)

	first := itinery.Itinery{
		Outbound: leg.Leg{ArrivalTime: baseTime.Add(8 * time.Hour)},
	}
	second := itinery.Itinery{
		Outbound: leg.Leg{DepartureTime: baseTime.Add(14 * time.Hour)}, // exactly 6h
	}

	// <= maxLayover
	if !validLayover(first, second, 1*time.Hour, 6*time.Hour) {
		t.Error("expected valid layover at exact maximum boundary")
	}
}

func TestValidLayover_BelowMinimum(t *testing.T) {
	baseTime := time.Date(2025, 6, 1, 8, 0, 0, 0, time.UTC)

	first := itinery.Itinery{
		Outbound: leg.Leg{ArrivalTime: baseTime.Add(8 * time.Hour)},
	}
	second := itinery.Itinery{
		Outbound: leg.Leg{DepartureTime: baseTime.Add(8*time.Hour + 30*time.Minute)}, // 30min
	}

	if validLayover(first, second, 1*time.Hour, 6*time.Hour) {
		t.Error("expected invalid layover below minimum")
	}
}

func TestValidLayover_AboveMaximum(t *testing.T) {
	baseTime := time.Date(2025, 6, 1, 8, 0, 0, 0, time.UTC)

	first := itinery.Itinery{
		Outbound: leg.Leg{ArrivalTime: baseTime.Add(8 * time.Hour)},
	}
	second := itinery.Itinery{
		Outbound: leg.Leg{DepartureTime: baseTime.Add(15 * time.Hour)}, // 7h
	}

	if validLayover(first, second, 1*time.Hour, 6*time.Hour) {
		t.Error("expected invalid layover above maximum")
	}
}

func TestValidLayover_NegativeLayover(t *testing.T) {
	baseTime := time.Date(2025, 6, 1, 8, 0, 0, 0, time.UTC)

	first := itinery.Itinery{
		Outbound: leg.Leg{ArrivalTime: baseTime.Add(12 * time.Hour)},
	}
	second := itinery.Itinery{
		Outbound: leg.Leg{DepartureTime: baseTime.Add(8 * time.Hour)}, // departs before arrival
	}

	if validLayover(first, second, 1*time.Hour, 6*time.Hour) {
		t.Error("expected invalid layover for negative duration")
	}
}

func TestValidLayover_ZeroLayover(t *testing.T) {
	baseTime := time.Date(2025, 6, 1, 8, 0, 0, 0, time.UTC)

	first := itinery.Itinery{
		Outbound: leg.Leg{ArrivalTime: baseTime.Add(8 * time.Hour)},
	}
	second := itinery.Itinery{
		Outbound: leg.Leg{DepartureTime: baseTime.Add(8 * time.Hour)}, // 0 layover
	}

	// 0 < 1h minimum → invalid
	if validLayover(first, second, 1*time.Hour, 6*time.Hour) {
		t.Error("expected invalid layover for zero duration with 1h minimum")
	}

	// With 0 minimum, 0 layover → valid (0 >= 0)
	if !validLayover(first, second, 0, 6*time.Hour) {
		t.Error("expected valid layover for zero duration with 0 minimum")
	}
}

func TestValidLayover_SameBounds(t *testing.T) {
	baseTime := time.Date(2025, 6, 1, 8, 0, 0, 0, time.UTC)

	first := itinery.Itinery{
		Outbound: leg.Leg{ArrivalTime: baseTime.Add(8 * time.Hour)},
	}
	second := itinery.Itinery{
		Outbound: leg.Leg{DepartureTime: baseTime.Add(11 * time.Hour)}, // 3h
	}

	// min == max == 3h → 3h >= 3h && 3h <= 3h → valid
	if !validLayover(first, second, 3*time.Hour, 3*time.Hour) {
		t.Error("expected valid layover when exactly matching same min/max bounds")
	}
}

// --- ConstructStop tests ---

func TestConstructStop_EmptyInputs(t *testing.T) {
	result := ConstructStop(nil, nil)
	if len(result) != 0 {
		t.Errorf("expected empty map, got %d entries", len(result))
	}
}

func TestConstructStop_NoMatchingAirports(t *testing.T) {
	baseTime := time.Date(2025, 6, 1, 8, 0, 0, 0, time.UTC)

	first := [][]itinery.Itinery{{
		createItinerary(
			createLeg("LHR", "JFK", baseTime, baseTime.Add(8*time.Hour)),
			createLeg("JFK", "LHR", baseTime.Add(72*time.Hour), baseTime.Add(80*time.Hour)),
			300.0,
		),
	}}
	second := [][]itinery.Itinery{{
		createItinerary(
			createLeg("CDG", "LAX", baseTime.Add(12*time.Hour), baseTime.Add(17*time.Hour)),
			createLeg("LAX", "CDG", baseTime.Add(48*time.Hour), baseTime.Add(53*time.Hour)),
			200.0,
		),
	}}

	result := ConstructStop(first, second)
	if len(result) != 0 {
		t.Errorf("expected empty map with no matching airports, got %d entries", len(result))
	}
}

func TestConstructStop_MatchingAirports(t *testing.T) {
	baseTime := time.Date(2025, 6, 1, 8, 0, 0, 0, time.UTC)

	first := [][]itinery.Itinery{{
		createItinerary(
			createLeg("LHR", "JFK", baseTime, baseTime.Add(8*time.Hour)),
			createLeg("JFK", "LHR", baseTime.Add(72*time.Hour), baseTime.Add(80*time.Hour)),
			300.0,
		),
	}}
	second := [][]itinery.Itinery{{
		createItinerary(
			createLeg("JFK", "LAX", baseTime.Add(12*time.Hour), baseTime.Add(17*time.Hour)),
			createLeg("LAX", "JFK", baseTime.Add(48*time.Hour), baseTime.Add(53*time.Hour)),
			200.0,
		),
	}}

	result := ConstructStop(first, second)
	if len(result) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result))
	}

	stop, exists := result["JFK"]
	if !exists {
		t.Fatal("expected JFK stop to exist")
	}
	if len(stop[0]) != 1 {
		t.Errorf("expected 1 first itinerary, got %d", len(stop[0]))
	}
	if len(stop[1]) != 1 {
		t.Errorf("expected 1 second itinerary, got %d", len(stop[1]))
	}
}

func TestConstructStop_MultipleStops(t *testing.T) {
	baseTime := time.Date(2025, 6, 1, 8, 0, 0, 0, time.UTC)

	first := [][]itinery.Itinery{
		{createItinerary(
			createLeg("LHR", "JFK", baseTime, baseTime.Add(8*time.Hour)),
			createLeg("JFK", "LHR", baseTime.Add(72*time.Hour), baseTime.Add(80*time.Hour)),
			300.0,
		)},
		{createItinerary(
			createLeg("LHR", "ORD", baseTime, baseTime.Add(9*time.Hour)),
			createLeg("ORD", "LHR", baseTime.Add(72*time.Hour), baseTime.Add(81*time.Hour)),
			350.0,
		)},
	}
	second := [][]itinery.Itinery{
		{createItinerary(
			createLeg("JFK", "LAX", baseTime.Add(12*time.Hour), baseTime.Add(17*time.Hour)),
			createLeg("LAX", "JFK", baseTime.Add(48*time.Hour), baseTime.Add(53*time.Hour)),
			200.0,
		)},
		{createItinerary(
			createLeg("ORD", "LAX", baseTime.Add(13*time.Hour), baseTime.Add(16*time.Hour)),
			createLeg("LAX", "ORD", baseTime.Add(49*time.Hour), baseTime.Add(52*time.Hour)),
			180.0,
		)},
	}

	result := ConstructStop(first, second)
	if len(result) != 2 {
		t.Errorf("expected 2 stops, got %d", len(result))
	}
}

func TestConstructStop_FirstWithoutSecond_Removed(t *testing.T) {
	baseTime := time.Date(2025, 6, 1, 8, 0, 0, 0, time.UTC)

	first := [][]itinery.Itinery{{
		createItinerary(
			createLeg("LHR", "JFK", baseTime, baseTime.Add(8*time.Hour)),
			createLeg("JFK", "LHR", baseTime.Add(72*time.Hour), baseTime.Add(80*time.Hour)),
			300.0,
		),
	}}
	// No matching second legs
	second := [][]itinery.Itinery{}

	result := ConstructStop(first, second)
	if len(result) != 0 {
		t.Errorf("expected 0 stops (first without second removed), got %d", len(result))
	}
}

func TestConstructStop_SecondWithoutFirst_Ignored(t *testing.T) {
	baseTime := time.Date(2025, 6, 1, 8, 0, 0, 0, time.UTC)

	first := [][]itinery.Itinery{}
	second := [][]itinery.Itinery{{
		createItinerary(
			createLeg("JFK", "LAX", baseTime.Add(12*time.Hour), baseTime.Add(17*time.Hour)),
			createLeg("LAX", "JFK", baseTime.Add(48*time.Hour), baseTime.Add(53*time.Hour)),
			200.0,
		),
	}}

	result := ConstructStop(first, second)
	if len(result) != 0 {
		t.Errorf("expected 0 stops (second without first ignored), got %d", len(result))
	}
}
