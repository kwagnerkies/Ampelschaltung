package controller

import (
	"testing"
	"time"

	"ampel/internal/light"
)

// Das ist die Zusage der Anzeige: die freigegebene Richtung zeigt ihre Restzeit, und diese
// Zahl springt hoch, sobald zwei Fahrzeuge dicht hintereinander ueber die Haltelinie fahren.
func TestDisplayedGreenJumpsWhenVehiclesFollow(t *testing.T) {
	h := newAdaptiveHarness(t)
	h.run(7 * time.Second)

	state := h.controller.State()
	if state.Phase != PhaseNS || state.Stage != StageGreen {
		t.Fatalf("Zustand %s, erwartet eine laufende Freigabe fuer Nord und Sued", state.Name())
	}
	before := h.controller.Snapshot(h.clk.Now())

	loads := []*load{{direction: light.North, stopLine: 5, upstream: []int{6, 13}, interval: 700 * time.Millisecond}}
	h.drive(loads, 2*time.Second)

	after := h.controller.Snapshot(h.clk.Now())
	if after.Following <= before.Following {
		t.Fatalf("%d Verlaengerungen, vorher %d", after.Following, before.Following)
	}
	if after.Green[light.North] <= before.Green[light.North] {
		t.Errorf("Restzeit fiel von %s auf %s, erwartet einen Sprung nach oben",
			before.Green[light.North], after.Green[light.North])
	}
	if after.Green[light.North] != after.Green[light.South] {
		t.Errorf("Nord zeigt %s, Sued %s, beide teilen sich die Freigabe",
			after.Green[light.North], after.Green[light.South])
	}
	if got := after.Green[light.East]; got != 8*time.Second {
		t.Errorf("die wartende Richtung zeigt %s, erwartet ihre Grundzeit von 8s", got)
	}
}

// Vereinzelter Verkehr verlaengert nicht: zwischen den Fahrzeugen liegt mehr als die
// Folgezeit.
func TestScatteredTrafficDoesNotExtend(t *testing.T) {
	h := newAdaptiveHarness(t)
	loads := []*load{
		{direction: light.North, stopLine: 5, upstream: []int{6, 13}, interval: 5 * time.Second},
		{direction: light.South, stopLine: 16, upstream: []int{20, 21}, interval: 5 * time.Second},
	}
	h.drive(loads, 8*time.Second)

	if got := h.controller.Snapshot(h.clk.Now()).Following; got != 0 {
		t.Errorf("%d Verlaengerungen bei vereinzeltem Verkehr, erwartet keine", got)
	}
}

// Die Anzeige darf den Regelkreis nicht beeinflussen.
func TestSnapshotIsSideEffectFree(t *testing.T) {
	h := newAdaptiveHarness(t)
	h.run(12 * time.Second)

	before := h.controller.State()
	first := h.controller.Snapshot(h.clk.Now())
	for i := 0; i < 50; i++ {
		h.controller.Snapshot(h.clk.Now())
	}
	if got := h.controller.State(); got != before {
		t.Errorf("Zustand nach vielen Abfragen %v, vorher %v", got, before)
	}
	if last := h.controller.Snapshot(h.clk.Now()); last.Green != first.Green {
		t.Errorf("Anzeige wanderte von %v auf %v", first.Green, last.Green)
	}
}

// Das Signalbild im Abtastwert muss zu dem passen, was die Lampen zeigen.
func TestSnapshotCarriesAspects(t *testing.T) {
	h := newAdaptiveHarness(t)
	h.run(12 * time.Second)
	snapshot := h.controller.Snapshot(h.clk.Now())
	if snapshot.Aspects != h.controller.State().Aspects() {
		t.Errorf("Signalbilder %v weichen vom Zustand ab", snapshot.Aspects)
	}
}
