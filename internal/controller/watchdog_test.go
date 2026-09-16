package controller

import (
	"testing"
	"time"

	"ampel/internal/light"
)

func TestWatchdogAcceptsRegularTicks(t *testing.T) {
	w := NewWatchdog(500*time.Millisecond, start)
	now := start
	for i := 0; i < 100; i++ {
		now = now.Add(50 * time.Millisecond)
		if gap, late := w.Kick(now); late {
			t.Fatalf("Takt %d nach %s als Ueberschreitung gemeldet", i, gap)
		}
	}
	now = now.Add(501 * time.Millisecond)
	if _, late := w.Kick(now); !late {
		t.Error("die Ueberschreitung blieb unbemerkt")
	}
}

// Bleibt der Regelkreis zu lange stehen, erzwingt der Watchdog den Notzustand: alle Lichter
// blinken gelb, und die Kreuzung gibt nicht mehr frei.
func TestWatchdogForcesFault(t *testing.T) {
	h := newHarness(t, 15*time.Second)
	h.run(4 * time.Second)
	if h.controller.State().Stage != StageGreen {
		t.Fatalf("Zustand %s, erwartet eine laufende Freigabe", h.controller.State().Name())
	}

	h.clk.Advance(DefaultWatchdog + 100*time.Millisecond)
	h.controller.Step(h.clk.Now())

	if got := h.controller.State().Phase; got != PhaseFault {
		t.Fatalf("Phase %s, erwartet Stoerung", got)
	}
	if len(h.observer.faults) != 1 {
		t.Fatalf("%d Stoerungen gemeldet, erwartet eine", len(h.observer.faults))
	}
	if !allShow(aspectsOf(h.mock.Pattern()), light.AspectYellow) {
		t.Error("der Notzustand zeigt kein Gelb auf allen Koepfen")
	}

	mark := len(h.mock.History())
	h.run(10 * time.Second)
	pulses := 0
	for _, pattern := range h.mock.History()[mark:] {
		aspects := aspectsOf(pattern)
		switch {
		case allShow(aspects, light.AspectYellow):
			pulses++
		case allShow(aspects, light.AspectOff):
		default:
			t.Fatalf("im Notzustand geschriebenes Muster %v", aspects)
		}
	}
	if pulses < 9 || pulses > 11 {
		t.Errorf("%d Gelbimpulse in zehn Sekunden, erwartet etwa zehn", pulses)
	}
	if got := h.controller.State().Phase; got != PhaseFault {
		t.Errorf("Phase %s, aus der Stoerung fuehrt nur ein Neustart", got)
	}
}
