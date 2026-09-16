package detector

import (
	"testing"
	"time"

	"ampel/internal/light"
)

var base = time.Date(2026, 4, 1, 7, 0, 0, 0, time.UTC)

var pins = [light.DirectionCount][]int{
	light.North: {5, 6, 13},
	light.East:  {19, 26, 12},
	light.South: {16, 20, 21},
	light.West:  {23, 24, 25},
}

func newDetector(t *testing.T) *Detector {
	t.Helper()
	d, err := New(pins)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return d
}

// Jede Flanke wird unmittelbar zum Ereignis. Entprellt hat der Kernel schon.
func TestFeedEmitsImmediately(t *testing.T) {
	d := newDetector(t)
	events, err := d.Feed(6, true, base)
	if err != nil {
		t.Fatalf("Feed: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("%d Ereignisse, erwartet eines", len(events))
	}
	got := events[0]
	if got.Direction != light.North || got.Index != 1 || !got.Occupied || !got.At.Equal(base) {
		t.Errorf("Ereignis %+v", got)
	}
}

// Derselbe Pegel zweimal ist keine Flanke.
func TestFeedIgnoresRepeatedLevel(t *testing.T) {
	d := newDetector(t)
	if _, err := d.Feed(19, true, base); err != nil {
		t.Fatalf("Feed: %v", err)
	}
	events, err := d.Feed(19, true, base.Add(time.Second))
	if err != nil {
		t.Fatalf("Feed: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("%d Ereignisse fuer denselben Pegel", len(events))
	}
}

func TestFeedRejectsUnknownPin(t *testing.T) {
	d := newDetector(t)
	if _, err := d.Feed(4, true, base); err == nil {
		t.Error("ein fremder Pin wurde angenommen")
	}
	if d.Knows(4) {
		t.Error("der Hauptschalter gilt als Sensor")
	}
	if !d.Knows(21) {
		t.Error("ein Sensorpin gilt als unbekannt")
	}
}

func TestNewRejectsDuplicatePin(t *testing.T) {
	doubled := pins
	doubled[light.West] = []int{23, 24, 5}
	if _, err := New(doubled); err == nil {
		t.Error("ein doppelt vergebener Pin wurde angenommen")
	}
}

// Nach dem Reset gilt jede Zufahrt als frei, die naechste Belegung ist wieder eine Flanke.
func TestResetForgetsLevels(t *testing.T) {
	d := newDetector(t)
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
