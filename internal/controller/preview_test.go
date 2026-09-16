package controller

import (
	"testing"
	"time"

	"ampel/internal/light"
)

// Das ist die Zusage der Anzeige: waehrend der Freigabe steht dort, wie lange sie noch
// dauert, und diese Zahl waechst mit jedem dicht folgenden Fahrzeug.
func TestDisplayedGreenGrowsWithFollowingVehicles(t *testing.T) {
	h := newAdaptiveHarness(t)
	loads := []*load{
		{direction: light.North, stopLine: 5, upstream: []int{6, 13}, interval: 700 * time.Millisecond},
		{direction: light.South, stopLine: 16, upstream: []int{20, 21}, interval: 700 * time.Millisecond},
	}
	h.drive(loads, 9*time.Second)

	state := h.controller.State()
	if state.Phase != PhaseNS || state.Stage != StageGreen {
		t.Fatalf("Zustand %s, erwartet eine laufende Freigabe fuer Nord und Sued", state.Name())
	}
	snapshot := h.controller.Snapshot(h.clk.Now())
	if snapshot.Following == 0 {
		t.Fatal("kein Fahrzeug wurde als dicht folgend gezaehlt")
	}
	if got := snapshot.Green[light.North]; got <= 8*time.Second {
		t.Errorf("Nord zeigt %s, erwartet mehr als die Grundzeit von 8s", got)
	}
	if snapshot.Green[light.North] != snapshot.Green[light.South] {
		t.Errorf("Nord zeigt %s, Sued %s, beide teilen sich die Freigabe",
			snapshot.Green[light.North], snapshot.Green[light.South])
	}
	// Die wartende Richtung zeigt ihre Grundzeit, sie hat noch nichts verlaengert.
	if got := snapshot.Green[light.East]; got != 8*time.Second {
		t.Errorf("Ost zeigt %s, erwartet die Grundzeit von 8s", got)
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
