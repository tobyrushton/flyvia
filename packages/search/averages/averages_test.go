package averages

import (
	"testing"
	"time"

	"github.com/tobyrushton/flyvia/packages/search/itinery"
	"github.com/tobyrushton/flyvia/packages/search/leg"
)

var baseTime = time.Date(2025, 6, 1, 8, 0, 0, 0, time.UTC)

func makeItin(
	outDepTime, outArrTime time.Time,
	inDepTime, inArrTime time.Time,
	outStops, inStops int,
	price float64,
) itinery.Itinery {
	return itinery.Itinery{
		Outbound: leg.Leg{
			DepartureTime: outDepTime,
			ArrivalTime:   outArrTime,
			Stops:         outStops,
		},
		Inbound: leg.Leg{
			DepartureTime: inDepTime,
			ArrivalTime:   inArrTime,
			Stops:         inStops,
		},
		Price: price,
	}
}

func TestCalculate_SingleItinerary(t *testing.T) {
	itins := []itinery.Itinery{
		makeItin(
			baseTime, baseTime.Add(5*time.Hour),
			baseTime.Add(7*24*time.Hour), baseTime.Add(7*24*time.Hour+5*time.Hour),
			0, 0, 400.0,
		),
	}

	avgDuration, avgStops, avgPrice := Calculate(itins)

	expectedDuration := 10 * time.Hour // 5h out + 5h in
	if avgDuration != expectedDuration {
		t.Errorf("expected duration %v, got %v", expectedDuration, avgDuration)
	}
	if avgStops != 0.0 {
		t.Errorf("expected 0 stops, got %f", avgStops)
	}
	if avgPrice != 400.0 {
		t.Errorf("expected price 400.0, got %f", avgPrice)
	}
}

func TestCalculate_MultipleItineraries(t *testing.T) {
	itins := []itinery.Itinery{
		makeItin(
			baseTime, baseTime.Add(5*time.Hour),
			baseTime.Add(7*24*time.Hour), baseTime.Add(7*24*time.Hour+5*time.Hour),
			0, 0, 300.0,
		),
		makeItin(
			baseTime, baseTime.Add(8*time.Hour),
			baseTime.Add(7*24*time.Hour), baseTime.Add(7*24*time.Hour+8*time.Hour),
			2, 2, 500.0,
		),
		makeItin(
			baseTime, baseTime.Add(6*time.Hour),
			baseTime.Add(7*24*time.Hour), baseTime.Add(7*24*time.Hour+6*time.Hour),
			1, 1, 400.0,
		),
	}

	avgDuration, avgStops, avgPrice := Calculate(itins)

	// Durations: 10h, 16h, 12h -> total = 38h -> avg = 38/3
	expectedDuration := time.Duration(float64(38*time.Hour) / 3.0)
	if avgDuration != expectedDuration {
		t.Errorf("expected duration %v, got %v", expectedDuration, avgDuration)
	}
	// Stops: (0+0)/2=0, (2+2)/2=2, (1+1)/2=1 -> total=3 -> avg=1.0
	if avgStops != 1.0 {
		t.Errorf("expected 1.0 stops, got %f", avgStops)
	}
	// Prices: 300+500+400=1200 -> avg=400
	if avgPrice != 400.0 {
		t.Errorf("expected price 400.0, got %f", avgPrice)
	}
}

func TestCalculate_AsymmetricStops(t *testing.T) {
	// Test with different outbound and inbound stops
	itins := []itinery.Itinery{
		makeItin(
			baseTime, baseTime.Add(5*time.Hour),
			baseTime.Add(7*24*time.Hour), baseTime.Add(7*24*time.Hour+5*time.Hour),
			1, 3, 400.0,
		),
	}

	_, avgStops, _ := Calculate(itins)

	// (1+3)/2 = 2 (integer division)
	expectedStops := 2.0
	if avgStops != expectedStops {
		t.Errorf("expected %f stops, got %f", expectedStops, avgStops)
	}
}

func TestCalculate_OddStopSum(t *testing.T) {
	// Test integer division behavior: (1+2)/2 = 1 (truncated)
	itins := []itinery.Itinery{
		makeItin(
			baseTime, baseTime.Add(5*time.Hour),
			baseTime.Add(7*24*time.Hour), baseTime.Add(7*24*time.Hour+5*time.Hour),
			1, 2, 400.0,
		),
	}

	_, avgStops, _ := Calculate(itins)

	expectedStops := 1.0
	if avgStops != expectedStops {
		t.Errorf("expected %f stops (integer division), got %f", expectedStops, avgStops)
	}
}

func TestCalculate_ZeroDuration(t *testing.T) {
	itins := []itinery.Itinery{
		makeItin(baseTime, baseTime, baseTime, baseTime, 0, 0, 100.0),
	}

	avgDuration, _, _ := Calculate(itins)

	if avgDuration != 0 {
		t.Errorf("expected 0 duration, got %v", avgDuration)
	}
}

func TestCalculate_ZeroPrices(t *testing.T) {
	itins := []itinery.Itinery{
		makeItin(baseTime, baseTime.Add(1*time.Hour), baseTime, baseTime.Add(1*time.Hour), 0, 0, 0.0),
	}

	_, _, avgPrice := Calculate(itins)

	if avgPrice != 0.0 {
		t.Errorf("expected 0 price, got %f", avgPrice)
	}
}

func TestCalculate_HighVolume(t *testing.T) {
	itins := make([]itinery.Itinery, 1000)
	for i := 0; i < 1000; i++ {
		itins[i] = makeItin(
			baseTime, baseTime.Add(5*time.Hour),
			baseTime.Add(7*24*time.Hour), baseTime.Add(7*24*time.Hour+5*time.Hour),
			1, 1, 500.0,
		)
	}

	avgDuration, avgStops, avgPrice := Calculate(itins)

	if avgDuration != 10*time.Hour {
		t.Errorf("expected 10h duration, got %v", avgDuration)
	}
	if avgStops != 1.0 {
		t.Errorf("expected 1.0 stops, got %f", avgStops)
	}
	if avgPrice != 500.0 {
		t.Errorf("expected 500.0 price, got %f", avgPrice)
	}
}

func TestCalculate_VaryingPrices(t *testing.T) {
	itins := []itinery.Itinery{
		makeItin(baseTime, baseTime.Add(1*time.Hour), baseTime, baseTime.Add(1*time.Hour), 0, 0, 100.0),
		makeItin(baseTime, baseTime.Add(1*time.Hour), baseTime, baseTime.Add(1*time.Hour), 0, 0, 900.0),
	}

	_, _, avgPrice := Calculate(itins)

	if avgPrice != 500.0 {
		t.Errorf("expected 500.0 average price, got %f", avgPrice)
	}
}

func TestCalculate_LongDuration(t *testing.T) {
	itins := []itinery.Itinery{
		makeItin(
			baseTime, baseTime.Add(48*time.Hour),
			baseTime.Add(7*24*time.Hour), baseTime.Add(9*24*time.Hour),
			3, 3, 1500.0,
		),
	}

	avgDuration, avgStops, _ := Calculate(itins)

	// 48h + 48h = 96h
	expectedDuration := 96 * time.Hour
	if avgDuration != expectedDuration {
		t.Errorf("expected duration %v, got %v", expectedDuration, avgDuration)
	}
	if avgStops != 3.0 {
		t.Errorf("expected 3.0 stops, got %f", avgStops)
	}
}
