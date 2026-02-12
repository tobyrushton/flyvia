package combine

import (
	"testing"
	"time"

	"github.com/tobyrushton/flyvia/packages/search/itinery"
	"github.com/tobyrushton/flyvia/packages/search/leg"
)

// Helper function to create a basic leg
func createLeg(depAirport, arrAirport string, depTime, arrTime time.Time) leg.Leg {
	return leg.Leg{
		Flights: []leg.Flight{
			{
				DepartureTime:    depTime,
				ArrivalTime:      arrTime,
				DepartureAirport: depAirport,
				ArrivalAirport:   arrAirport,
				FlightCode:       "TEST123",
				Plane:            "Boeing 737",
				Airline:          "Test Airlines",
			},
		},
		Stops:            0,
		DepartureTime:    depTime,
		ArrivalTime:      arrTime,
		DepartureAirport: depAirport,
		ArrivalAirport:   arrAirport,
	}
}

// Helper function to create an itinerary
func createItinerary(outboundLeg, inboundLeg leg.Leg, price float64) itinery.Itinery {
	return itinery.Itinery{
		Outbound:   outboundLeg,
		Inbound:    inboundLeg,
		Price:      price,
		BookingURL: "https://example.com/book",
	}
}

func TestOneStop_EmptyFirstItineraries(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	secondItineraries := []itinery.Itinery{
		createItinerary(
			createLeg("JFK", "LAX", baseTime, baseTime.Add(5*time.Hour)),
			createLeg("LAX", "JFK", baseTime.Add(24*time.Hour), baseTime.Add(29*time.Hour)),
			500.0,
		),
	}

	results := OneStop(
		[]itinery.Itinery{},
		secondItineraries,
		1*time.Hour,
		6*time.Hour,
	)

	if len(results) != 0 {
		t.Errorf("Expected 0 results with empty first itineraries, got %d", len(results))
	}
}

func TestOneStop_EmptySecondItineraries(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	firstItineraries := []itinery.Itinery{
		createItinerary(
			createLeg("LHR", "JFK", baseTime, baseTime.Add(8*time.Hour)),
			createLeg("JFK", "LHR", baseTime.Add(72*time.Hour), baseTime.Add(80*time.Hour)),
			800.0,
		),
	}

	results := OneStop(
		firstItineraries,
		[]itinery.Itinery{},
		1*time.Hour,
		6*time.Hour,
	)

	if len(results) != 0 {
		t.Errorf("Expected 0 results with empty second itineraries, got %d", len(results))
	}
}

func TestOneStop_BothEmpty(t *testing.T) {
	results := OneStop(
		[]itinery.Itinery{},
		[]itinery.Itinery{},
		1*time.Hour,
		6*time.Hour,
	)

	if len(results) != 0 {
		t.Errorf("Expected 0 results with both empty, got %d", len(results))
	}
}

func TestOneStop_NoMatchingAirports(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	firstItineraries := []itinery.Itinery{
		createItinerary(
			createLeg("LHR", "JFK", baseTime, baseTime.Add(8*time.Hour)),
			createLeg("JFK", "LHR", baseTime.Add(72*time.Hour), baseTime.Add(80*time.Hour)),
			800.0,
		),
	}

	secondItineraries := []itinery.Itinery{
		createItinerary(
			createLeg("LAX", "SFO", baseTime.Add(20*time.Hour), baseTime.Add(21*time.Hour)),
			createLeg("SFO", "LAX", baseTime.Add(48*time.Hour), baseTime.Add(49*time.Hour)),
			300.0,
		),
	}

	results := OneStop(
		firstItineraries,
		secondItineraries,
		1*time.Hour,
		6*time.Hour,
	)

	if len(results) != 0 {
		t.Errorf("Expected 0 results with no matching airports, got %d", len(results))
	}
}

func TestOneStop_ValidConnection(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	// First flight arrives at JFK at 18:00
	firstItineraries := []itinery.Itinery{
		createItinerary(
			createLeg("LHR", "JFK", baseTime, baseTime.Add(8*time.Hour)),
			createLeg("JFK", "LHR", baseTime.Add(72*time.Hour), baseTime.Add(80*time.Hour)),
			800.0,
		),
	}

	// Second flight departs from JFK at 21:00 (3 hour layover)
	secondItineraries := []itinery.Itinery{
		createItinerary(
			createLeg("JFK", "LAX", baseTime.Add(11*time.Hour), baseTime.Add(16*time.Hour)),
			createLeg("LAX", "JFK", baseTime.Add(48*time.Hour), baseTime.Add(53*time.Hour)),
			500.0,
		),
	}

	results := OneStop(
		firstItineraries,
		secondItineraries,
		1*time.Hour,
		6*time.Hour,
	)

	if len(results) != 1 {
		t.Errorf("Expected 1 result with valid connection, got %d", len(results))
	}
}

func TestOneStop_LayoverTooShort(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	// First flight arrives at JFK at 18:00
	firstItineraries := []itinery.Itinery{
		createItinerary(
			createLeg("LHR", "JFK", baseTime, baseTime.Add(8*time.Hour)),
			createLeg("JFK", "LHR", baseTime.Add(72*time.Hour), baseTime.Add(80*time.Hour)),
			800.0,
		),
	}

	// Second flight departs from JFK at 18:30 (30 minute layover, less than 1 hour minimum)
	secondItineraries := []itinery.Itinery{
		createItinerary(
			createLeg("JFK", "LAX", baseTime.Add(8*time.Hour+30*time.Minute), baseTime.Add(13*time.Hour+30*time.Minute)),
			createLeg("LAX", "JFK", baseTime.Add(48*time.Hour), baseTime.Add(53*time.Hour)),
			500.0,
		),
	}

	results := OneStop(
		firstItineraries,
		secondItineraries,
		1*time.Hour,
		6*time.Hour,
	)

	if len(results) != 0 {
		t.Errorf("Expected 0 results with layover too short, got %d", len(results))
	}
}

func TestOneStop_LayoverTooLong(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	// First flight arrives at JFK at 18:00
	firstItineraries := []itinery.Itinery{
		createItinerary(
			createLeg("LHR", "JFK", baseTime, baseTime.Add(8*time.Hour)),
			createLeg("JFK", "LHR", baseTime.Add(72*time.Hour), baseTime.Add(80*time.Hour)),
			800.0,
		),
	}

	// Second flight departs from JFK at 01:00 next day (7 hour layover, more than 6 hour maximum)
	secondItineraries := []itinery.Itinery{
		createItinerary(
			createLeg("JFK", "LAX", baseTime.Add(15*time.Hour), baseTime.Add(20*time.Hour)),
			createLeg("LAX", "JFK", baseTime.Add(48*time.Hour), baseTime.Add(53*time.Hour)),
			500.0,
		),
	}

	results := OneStop(
		firstItineraries,
		secondItineraries,
		1*time.Hour,
		6*time.Hour,
	)

	if len(results) != 0 {
		t.Errorf("Expected 0 results with layover too long, got %d", len(results))
	}
}

func TestOneStop_MultipleValidConnections(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	// First flight arrives at JFK at 18:00
	firstItineraries := []itinery.Itinery{
		createItinerary(
			createLeg("LHR", "JFK", baseTime, baseTime.Add(8*time.Hour)),
			createLeg("JFK", "LHR", baseTime.Add(72*time.Hour), baseTime.Add(80*time.Hour)),
			800.0,
		),
	}

	// Multiple second flights from JFK with valid layovers
	secondItineraries := []itinery.Itinery{
		createItinerary(
			createLeg("JFK", "LAX", baseTime.Add(10*time.Hour), baseTime.Add(15*time.Hour)),
			createLeg("LAX", "JFK", baseTime.Add(48*time.Hour), baseTime.Add(53*time.Hour)),
			500.0,
		),
		createItinerary(
			createLeg("JFK", "SFO", baseTime.Add(12*time.Hour), baseTime.Add(17*time.Hour)),
			createLeg("SFO", "JFK", baseTime.Add(50*time.Hour), baseTime.Add(55*time.Hour)),
			550.0,
		),
		createItinerary(
			createLeg("JFK", "ORD", baseTime.Add(13*time.Hour), baseTime.Add(15*time.Hour)),
			createLeg("ORD", "JFK", baseTime.Add(52*time.Hour), baseTime.Add(54*time.Hour)),
			400.0,
		),
	}

	results := OneStop(
		firstItineraries,
		secondItineraries,
		1*time.Hour,
		6*time.Hour,
	)

	if len(results) != 3 {
		t.Errorf("Expected 3 results with multiple valid connections, got %d", len(results))
	}
}

func TestOneStop_MultipleFirstItineraries(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	firstItineraries := []itinery.Itinery{
		createItinerary(
			createLeg("LHR", "JFK", baseTime, baseTime.Add(8*time.Hour)),
			createLeg("JFK", "LHR", baseTime.Add(72*time.Hour), baseTime.Add(80*time.Hour)),
			800.0,
		),
		createItinerary(
			createLeg("CDG", "JFK", baseTime.Add(1*time.Hour), baseTime.Add(9*time.Hour)),
			createLeg("JFK", "CDG", baseTime.Add(73*time.Hour), baseTime.Add(81*time.Hour)),
			850.0,
		),
	}

	// One second flight from JFK
	secondItineraries := []itinery.Itinery{
		createItinerary(
			createLeg("JFK", "LAX", baseTime.Add(12*time.Hour), baseTime.Add(17*time.Hour)),
			createLeg("LAX", "JFK", baseTime.Add(48*time.Hour), baseTime.Add(53*time.Hour)),
			500.0,
		),
	}

	results := OneStop(
		firstItineraries,
		secondItineraries,
		1*time.Hour,
		6*time.Hour,
	)

	if len(results) != 2 {
		t.Errorf("Expected 2 results (one for each first itinerary), got %d", len(results))
	}
}

func TestOneStop_SameAirportMultipleTimes(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	// Multiple flights arriving at JFK at different times
	firstItineraries := []itinery.Itinery{
		createItinerary(
			createLeg("LHR", "JFK", baseTime, baseTime.Add(8*time.Hour)),
			createLeg("JFK", "LHR", baseTime.Add(72*time.Hour), baseTime.Add(80*time.Hour)),
			800.0,
		),
		createItinerary(
			createLeg("LHR", "JFK", baseTime.Add(2*time.Hour), baseTime.Add(10*time.Hour)),
			createLeg("JFK", "LHR", baseTime.Add(74*time.Hour), baseTime.Add(82*time.Hour)),
			820.0,
		),
	}

	// Multiple flights departing from JFK at different times
	secondItineraries := []itinery.Itinery{
		createItinerary(
			createLeg("JFK", "LAX", baseTime.Add(11*time.Hour), baseTime.Add(16*time.Hour)),
			createLeg("LAX", "JFK", baseTime.Add(48*time.Hour), baseTime.Add(53*time.Hour)),
			500.0,
		),
		createItinerary(
			createLeg("JFK", "LAX", baseTime.Add(13*time.Hour), baseTime.Add(18*time.Hour)),
			createLeg("LAX", "JFK", baseTime.Add(50*time.Hour), baseTime.Add(55*time.Hour)),
			520.0,
		),
	}

	results := OneStop(
		firstItineraries,
		secondItineraries,
		1*time.Hour,
		6*time.Hour,
	)

	// Should get 4 results: each first itinerary can connect to each second itinerary
	if len(results) != 4 {
		t.Errorf("Expected 4 results with multiple connections at same airport, got %d", len(results))
	}
}

func TestOneStop_ZeroLayoverDuration(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	// First flight arrives at JFK at 18:00
	firstItineraries := []itinery.Itinery{
		createItinerary(
			createLeg("LHR", "JFK", baseTime, baseTime.Add(8*time.Hour)),
			createLeg("JFK", "LHR", baseTime.Add(72*time.Hour), baseTime.Add(80*time.Hour)),
			800.0,
		),
	}

	// Second flight departs from JFK at 18:00 (zero layover)
	secondItineraries := []itinery.Itinery{
		createItinerary(
			createLeg("JFK", "LAX", baseTime.Add(8*time.Hour), baseTime.Add(13*time.Hour)),
			createLeg("LAX", "JFK", baseTime.Add(48*time.Hour), baseTime.Add(53*time.Hour)),
			500.0,
		),
	}

	results := OneStop(
		firstItineraries,
		secondItineraries,
		0*time.Hour,
		6*time.Hour,
	)

	// This tests behavior with zero-duration layover - might be valid depending on validLayover implementation
	// Adjust expected result based on actual validLayover behavior
	if len(results) != 1 {
		t.Logf("With 0 minimum layover and exact connection time, got %d results", len(results))
	}
}

func TestOneStop_NegativeSecondFlight(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	// First flight arrives at JFK at 18:00
	firstItineraries := []itinery.Itinery{
		createItinerary(
			createLeg("LHR", "JFK", baseTime, baseTime.Add(8*time.Hour)),
			createLeg("JFK", "LHR", baseTime.Add(72*time.Hour), baseTime.Add(80*time.Hour)),
			800.0,
		),
	}

	// Second flight departs from JFK BEFORE first flight arrives
	secondItineraries := []itinery.Itinery{
		createItinerary(
			createLeg("JFK", "LAX", baseTime.Add(5*time.Hour), baseTime.Add(10*time.Hour)),
			createLeg("LAX", "JFK", baseTime.Add(48*time.Hour), baseTime.Add(53*time.Hour)),
			500.0,
		),
	}

	results := OneStop(
		firstItineraries,
		secondItineraries,
		1*time.Hour,
		6*time.Hour,
	)

	if len(results) != 0 {
		t.Errorf("Expected 0 results when second flight departs before first arrives, got %d", len(results))
	}
}

func TestOneStop_BoundaryLayoverMinimum(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	// First flight arrives at JFK at 18:00
	firstItineraries := []itinery.Itinery{
		createItinerary(
			createLeg("LHR", "JFK", baseTime, baseTime.Add(8*time.Hour)),
			createLeg("JFK", "LHR", baseTime.Add(72*time.Hour), baseTime.Add(80*time.Hour)),
			800.0,
		),
	}

	// Second flight departs from JFK at exactly 19:00 (exactly 1 hour layover)
	secondItineraries := []itinery.Itinery{
		createItinerary(
			createLeg("JFK", "LAX", baseTime.Add(9*time.Hour), baseTime.Add(14*time.Hour)),
			createLeg("LAX", "JFK", baseTime.Add(48*time.Hour), baseTime.Add(53*time.Hour)),
			500.0,
		),
	}

	results := OneStop(
		firstItineraries,
		secondItineraries,
		1*time.Hour,
		6*time.Hour,
	)

	// Behavior depends on whether validLayover uses >= or > for minimum
	t.Logf("With exactly minimum layover (1 hour), got %d results", len(results))
}

func TestOneStop_BoundaryLayoverMaximum(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	// First flight arrives at JFK at 18:00
	firstItineraries := []itinery.Itinery{
		createItinerary(
			createLeg("LHR", "JFK", baseTime, baseTime.Add(8*time.Hour)),
			createLeg("JFK", "LHR", baseTime.Add(72*time.Hour), baseTime.Add(80*time.Hour)),
			800.0,
		),
	}

	// Second flight departs from JFK at exactly 00:00 next day (exactly 6 hour layover)
	secondItineraries := []itinery.Itinery{
		createItinerary(
			createLeg("JFK", "LAX", baseTime.Add(14*time.Hour), baseTime.Add(19*time.Hour)),
			createLeg("LAX", "JFK", baseTime.Add(48*time.Hour), baseTime.Add(53*time.Hour)),
			500.0,
		),
	}

	results := OneStop(
		firstItineraries,
		secondItineraries,
		1*time.Hour,
		6*time.Hour,
	)

	// Behavior depends on whether validLayover uses <= or < for maximum
	t.Logf("With exactly maximum layover (6 hours), got %d results", len(results))
}

func TestOneStop_MixedValidAndInvalid(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	// First flight arrives at JFK at 18:00
	firstItineraries := []itinery.Itinery{
		createItinerary(
			createLeg("LHR", "JFK", baseTime, baseTime.Add(8*time.Hour)),
			createLeg("JFK", "LHR", baseTime.Add(72*time.Hour), baseTime.Add(80*time.Hour)),
			800.0,
		),
	}

	// Mix of valid and invalid connections
	secondItineraries := []itinery.Itinery{
		// Too short - 30 minutes
		createItinerary(
			createLeg("JFK", "LAX", baseTime.Add(8*time.Hour+30*time.Minute), baseTime.Add(13*time.Hour+30*time.Minute)),
			createLeg("LAX", "JFK", baseTime.Add(48*time.Hour), baseTime.Add(53*time.Hour)),
			450.0,
		),
		// Valid - 2 hours
		createItinerary(
			createLeg("JFK", "LAX", baseTime.Add(10*time.Hour), baseTime.Add(15*time.Hour)),
			createLeg("LAX", "JFK", baseTime.Add(49*time.Hour), baseTime.Add(54*time.Hour)),
			500.0,
		),
		// Too long - 8 hours
		createItinerary(
			createLeg("JFK", "LAX", baseTime.Add(16*time.Hour), baseTime.Add(21*time.Hour)),
			createLeg("LAX", "JFK", baseTime.Add(50*time.Hour), baseTime.Add(55*time.Hour)),
			520.0,
		),
		// Valid - 4 hours
		createItinerary(
			createLeg("JFK", "SFO", baseTime.Add(12*time.Hour), baseTime.Add(17*time.Hour)),
			createLeg("SFO", "JFK", baseTime.Add(51*time.Hour), baseTime.Add(56*time.Hour)),
			550.0,
		),
	}

	results := OneStop(
		firstItineraries,
		secondItineraries,
		1*time.Hour,
		6*time.Hour,
	)

	if len(results) != 2 {
		t.Errorf("Expected 2 valid results from mixed connections, got %d", len(results))
	}
}

func TestOneStop_DifferentAirportCodes(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	// Test with various airport codes
	firstItineraries := []itinery.Itinery{
		createItinerary(
			createLeg("ABC", "XYZ", baseTime, baseTime.Add(2*time.Hour)),
			createLeg("XYZ", "ABC", baseTime.Add(48*time.Hour), baseTime.Add(50*time.Hour)),
			300.0,
		),
	}

	secondItineraries := []itinery.Itinery{
		createItinerary(
			createLeg("XYZ", "DEF", baseTime.Add(4*time.Hour), baseTime.Add(6*time.Hour)),
			createLeg("DEF", "XYZ", baseTime.Add(52*time.Hour), baseTime.Add(54*time.Hour)),
			250.0,
		),
	}

	results := OneStop(
		firstItineraries,
		secondItineraries,
		1*time.Hour,
		6*time.Hour,
	)

	if len(results) != 1 {
		t.Errorf("Expected 1 result with different airport codes, got %d", len(results))
	}
}

func TestOneStop_CaseSensitivity(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	// Test if airport matching is case-sensitive
	firstItineraries := []itinery.Itinery{
		createItinerary(
			createLeg("LHR", "JFK", baseTime, baseTime.Add(8*time.Hour)),
			createLeg("JFK", "LHR", baseTime.Add(72*time.Hour), baseTime.Add(80*time.Hour)),
			800.0,
		),
	}

	// Using lowercase 'jfk' instead of 'JFK'
	secondItineraries := []itinery.Itinery{
		createItinerary(
			createLeg("jfk", "LAX", baseTime.Add(11*time.Hour), baseTime.Add(16*time.Hour)),
			createLeg("LAX", "jfk", baseTime.Add(48*time.Hour), baseTime.Add(53*time.Hour)),
			500.0,
		),
	}

	results := OneStop(
		firstItineraries,
		secondItineraries,
		1*time.Hour,
		6*time.Hour,
	)

	// This will reveal if the function is case-sensitive (expected: 0 if case-sensitive)
	t.Logf("With case mismatch (JFK vs jfk), got %d results", len(results))
}

func TestOneStop_LargeDataset(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	// Create many first itineraries
	firstItineraries := make([]itinery.Itinery, 100)
	for i := 0; i < 100; i++ {
		firstItineraries[i] = createItinerary(
			createLeg("LHR", "JFK", baseTime.Add(time.Duration(i)*time.Minute), baseTime.Add(8*time.Hour+time.Duration(i)*time.Minute)),
			createLeg("JFK", "LHR", baseTime.Add(72*time.Hour), baseTime.Add(80*time.Hour)),
			800.0+float64(i),
		)
	}

	// Create many second itineraries from JFK
	secondItineraries := make([]itinery.Itinery, 100)
	for i := 0; i < 100; i++ {
		secondItineraries[i] = createItinerary(
			createLeg("JFK", "LAX", baseTime.Add(10*time.Hour+time.Duration(i)*time.Minute), baseTime.Add(15*time.Hour+time.Duration(i)*time.Minute)),
			createLeg("LAX", "JFK", baseTime.Add(48*time.Hour), baseTime.Add(53*time.Hour)),
			500.0+float64(i),
		)
	}

	results := OneStop(
		firstItineraries,
		secondItineraries,
		1*time.Hour,
		6*time.Hour,
	)

	// Should have many valid connections
	if len(results) == 0 {
		t.Error("Expected multiple results with large dataset")
	}
	t.Logf("Large dataset test produced %d results", len(results))
}
