package traffic

import (
	"testing"
	"time"

	"ampel/internal/detector"
	"ampel/internal/light"
)

var base = time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)

func newApproach(t *testing.T) *Approach {
	t.Helper()
	queue, err := detector.NewQueue(map[int]int{0: 0, 1: 1, 2: 3, 3: 6}, 3)
	if err != nil {
		t.Fatalf("NewQueue: %v", err)
	}
	approach, err := NewApproach(light.North, 3, queue)
	if err != nil {
		t.Fatalf("NewApproach: %v", err)
	}
	return approach
}

func apply(t *testing.T, a *Approach, index int, occupied bool, offset time.Duration, phase int) (Departure, bool) {
	t.Helper()
	return a.Apply(detector.SensorEvent{
		Direction: a.Direction(),
		Index:     index,
		Occupied:  occupied,
		At:        base.Add(offset),
	}, phase)
}

// Ein Fahrzeug, das ueber alle drei Kontakte rollt, an der Linie haelt und bei Gruen
// abfaehrt, muss eine plausible Wartezeit ergeben.
func TestVehicleFromArrivalToDeparture(t *testing.T) {
	a := newApproach(t)

	apply(t, a, 2, true, 0, 1)
	apply(t, a, 2, false, 400*time.Millisecond, 1)
	apply(t, a, 1, true, 500*time.Millisecond, 1)
	apply(t, a, 1, false, 900*time.Millisecond, 1)
	apply(t, a, 0, true, time.Second, 1)

	if a.Waiting() != 1 {
		t.Fatalf("%d wartende Fahrzeuge, erwartet eines", a.Waiting())
	}
	if !a.AtStopLine() {
		t.Error("Haltelinie gilt als frei")
	}

	departure, ok := apply(t, a, 0, false, 12*time.Second, 1)
	if !ok {
		t.Fatal("die Abfahrt wurde nicht erkannt")
	}
	if departure.Wait != 12*time.Second {
		t.Errorf("Wartezeit %s, erwartet 12s", departure.Wait)
	}
	if departure.Direction != light.North || departure.Arrival.Phase != 1 {
		t.Errorf("Abfahrt %+v, erwartet Nord in Phase 1", departure)
	}
	if a.Waiting() != 0 {
		t.Errorf("%d wartende Fahrzeuge nach der Abfahrt, erwartet keines", a.Waiting())
	}
}

// Ein von Hand direkt auf die Linie gesetztes Fahrzeug wird nicht ueber den hinteren Sensor
// erfasst. Dann beginnt die Messung an der Linie, statt das Fahrzeug zu verlieren.
func TestVehiclePlacedAtStopLineCounts(t *testing.T) {
	a := newApproach(t)

	apply(t, a, 0, true, 0, 2)
	if a.Waiting() != 1 {
		t.Fatalf("%d wartende Fahrzeuge, erwartet eines", a.Waiting())
	}
	departure, ok := apply(t, a, 0, false, 3*time.Second, 2)
	if !ok || departure.Wait != 3*time.Second {
		t.Errorf("Abfahrt %+v, %v, erwartet 3s Wartezeit", departure, ok)
	}
}

// Rollt ein zweites Fahrzeug nach, darf die Ankunft an der Linie nicht doppelt zaehlen.
func TestQueueOfTwoVehiclesKeepsOrder(t *testing.T) {
	a := newApproach(t)

	apply(t, a, 2, true, 0, 1)
	apply(t, a, 2, false, time.Second, 1)
	apply(t, a, 0, true, 2*time.Second, 1)
	apply(t, a, 2, true, 3*time.Second, 1)

	if a.Waiting() != 2 {
		t.Fatalf("%d wartende Fahrzeuge, erwartet zwei", a.Waiting())
	}

	first, ok := apply(t, a, 0, false, 10*time.Second, 1)
	if !ok || first.Wait != 10*time.Second {
		t.Fatalf("erste Abfahrt %+v, erwartet 10s Wartezeit", first)
	}
	apply(t, a, 0, true, 11*time.Second, 1)
	second, ok := apply(t, a, 0, false, 14*time.Second, 1)
	if !ok || second.Wait != 11*time.Second {
		t.Errorf("zweite Abfahrt %+v, erwartet 11s Wartezeit", second)
	}
}

func TestQueueLengthAndArrivalQueue(t *testing.T) {
	a := newApproach(t)

	apply(t, a, 0, true, 0, 0)
	if got := a.QueueLength(); got != 1 {
		t.Errorf("Rueckstau %d, erwartet 1", got)
	}
	apply(t, a, 2, true, time.Second, 0)
	if got := a.QueueLength(); got != 6 {
		t.Errorf("Rueckstau %d, erwartet 6", got)
	}
}
func TestResetClearsEverything(t *testing.T) {
	a := newApproach(t)
	apply(t, a, 2, true, 0, 0)

	a.Reset()
	if a.Waiting() != 0 || a.QueueLength() != 0 {
		t.Errorf("nach dem Reset: %d wartend, Rueckstau %d", a.Waiting(), a.QueueLength())
	}
}

func TestNewApproachRejectsBadParameters(t *testing.T) {
	queue, err := detector.NewQueue(map[int]int{0: 0, 1: 1}, 1)
	if err != nil {
		t.Fatalf("NewQueue: %v", err)
	}
	if _, err := NewApproach(light.North, 0, queue); err == nil {
		t.Error("Zufahrt ohne Sensoren wurde angenommen")
	}
}
