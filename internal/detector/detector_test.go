package detector

import (
	"testing"
	"time"

	"ampel/internal/light"
)

var (
	base = time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	pins = [light.DirectionCount][]int{
		light.North: {5, 6, 13},
		light.East:  {19, 26, 12},
		light.South: {16, 20, 21},
		light.West:  {23, 24, 25},
	}
)

func newDetector(t *testing.T, debounce time.Duration) *Detector {
	t.Helper()
	d, err := New(pins, debounce)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return d
}

// Der Zeitstempel des Ereignisses ist der der Flanke, nicht der der Uebernahme. Sonst
// verschiebt die Ruhezeit jede Wartezeitmessung.
func TestFeedEmitsAfterQuietTime(t *testing.T) {
	d := newDetector(t, 15*time.Millisecond)

	events, err := d.Feed(5, true, base)
	if err != nil {
		t.Fatalf("Feed: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("%d Ereignisse sofort, erwartet keines", len(events))
	}

	events = d.Tick(base.Add(15 * time.Millisecond))
	if len(events) != 1 {
		t.Fatalf("%d Ereignisse nach der Ruhezeit, erwartet eines", len(events))
	}
	want := SensorEvent{Direction: light.North, Index: 0, Occupied: true, At: base}
	if events[0] != want {
		t.Errorf("Ereignis %+v, erwartet %+v", events[0], want)
	}
}

// Ein prellender Kontakt darf kein Phantomfahrzeug erzeugen. Erst wenn der Pegel steht,
// entsteht genau ein Ereignis.
func TestBouncingContactProducesSingleEvent(t *testing.T) {
	d := newDetector(t, 15*time.Millisecond)

	// Ungerade Anzahl Wechsel, damit der Kontakt am Ende geschlossen bleibt.
	level := true
	at := base
	for i := 0; i < 11; i++ {
		events, err := d.Feed(13, level, at)
		if err != nil {
			t.Fatalf("Feed: %v", err)
		}
		if len(events) != 0 {
			t.Fatalf("prellen bei %s erzeugte %d Ereignisse", at.Sub(base), len(events))
		}
		level = !level
		at = at.Add(5 * time.Millisecond)
	}

	events := d.Tick(at.Add(20 * time.Millisecond))
	if len(events) != 1 {
		t.Fatalf("%d Ereignisse nach dem Prellen, erwartet eines", len(events))
	}
	if !events[0].Occupied || events[0].Index != 2 || events[0].Direction != light.North {
		t.Errorf("Ereignis %+v, erwartet Nord Sensor 2 belegt", events[0])
	}
}

func TestFeedWithoutQuietTimeIsImmediate(t *testing.T) {
	d := newDetector(t, 0)

	events, err := d.Feed(19, true, base)
	if err != nil {
		t.Fatalf("Feed: %v", err)
	}
	if len(events) != 1 || events[0].Direction != light.East {
		t.Fatalf("Ereignisse %+v, erwartet eines fuer Ost", events)
	}
}

func TestFeedIgnoresRepeatedLevel(t *testing.T) {
	d := newDetector(t, 0)
	if _, err := d.Feed(21, true, base); err != nil {
		t.Fatalf("Feed: %v", err)
	}
	events, err := d.Feed(21, true, base.Add(time.Second))
	if err != nil {
		t.Fatalf("Feed: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("%d Ereignisse bei gleichem Pegel, erwartet keines", len(events))
	}
}

func TestFeedRejectsUnknownPin(t *testing.T) {
	d := newDetector(t, 0)
	if _, err := d.Feed(99, true, base); err == nil {
		t.Error("unbekannter Pin wurde angenommen")
	}
}

func TestNewRejectsDuplicatePin(t *testing.T) {
	doubled := pins
	doubled[light.West] = []int{5, 24, 25}
	if _, err := New(doubled, 0); err == nil {
		t.Error("doppelt zugeordneter Pin wurde angenommen")
	}
}

func TestResetForgetsLevels(t *testing.T) {
	d := newDetector(t, 0)
	if _, err := d.Feed(16, true, base); err != nil {
		t.Fatalf("Feed: %v", err)
	}
	d.Reset()

	events, err := d.Feed(16, true, base.Add(time.Second))
	if err != nil {
		t.Fatalf("Feed: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("%d Ereignisse nach dem Reset, erwartet eines", len(events))
	}
}
