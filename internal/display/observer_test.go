package display

import (
	"testing"
	"time"

	"ampel/internal/controller"
	"ampel/internal/detector"
	"ampel/internal/light"
)

func snapshot(east time.Duration) controller.Snapshot {
	s := controller.Snapshot{}
	s.Aspects = [light.DirectionCount]light.Aspect{
		light.North: light.AspectGreen,
		light.East:  light.AspectRed,
		light.South: light.AspectGreen,
		light.West:  light.AspectRed,
	}
	s.Green = [light.DirectionCount]time.Duration{
		light.North: 18 * time.Second,
		light.East:  east,
		light.South: 18 * time.Second,
		light.West:  east,
	}
	return s
}

// Ein Sensorereignis zeichnet sofort neu, ohne auf den naechsten Abtastwert zu warten.
func TestSensorEventRedrawsImmediately(t *testing.T) {
	canvas := newFake()
	current := snapshot(7 * time.Second)
	observer := NewObserver(New(canvas), func(time.Time) controller.Snapshot { return current }, nil)

	observer.Sample(time.Time{}, current)
	canvas.fills = nil

	current = snapshot(12 * time.Second)
	observer.SensorChanged(detector.SensorEvent{Direction: light.East, Occupied: true})
	if len(canvas.fills) == 0 {
		t.Fatal("das Sensorereignis zeichnete nichts neu")
	}
	if got, want := len(canvas.fills), 2*2*7; got != want {
		t.Errorf("%d Zeichenbefehle, erwartet %d fuer Ost und West", got, want)
	}
}

// Sekunden werden gerundet, nicht abgeschnitten: 7,6 Sekunden sind eine Acht.
func TestSecondsAreRounded(t *testing.T) {
	canvas := newFake()
	screen := New(canvas)
	observer := NewObserver(screen, nil, nil)
	observer.Sample(time.Time{}, snapshot(7600*time.Millisecond))
	if got := screen.last[light.East].Seconds; got != 8 {
		t.Errorf("7,6 Sekunden werden als %d angezeigt", got)
	}
}

// Ohne Quelle darf ein Sensorereignis nicht abstuerzen.
func TestObserverWithoutSourceIsSilent(t *testing.T) {
	observer := NewObserver(New(newFake()), nil, nil)
	observer.SensorChanged(detector.SensorEvent{Direction: light.North})
	observer.PhaseChanged(time.Time{}, controller.State{}, "adaptiv")
}
