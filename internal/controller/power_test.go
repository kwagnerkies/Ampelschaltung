package controller

import (
	"testing"
	"time"

	"ampel/internal/light"
)

const powerPin = 4

// newPowerHarness startet mit eingeschalteter Anlage.
func newPowerHarness(t *testing.T) *harness {
	t.Helper()
	return newHarness(t, 15*time.Second, func(setup *Setup) {
		setup.Power = &Power{Pin: powerPin, On: true}
	})
}

func (h *harness) flip(on bool) {
	h.controller.Feed(Input{Pin: powerPin, Active: on, Time: h.clk.Now()})
	h.run(300 * time.Millisecond)
}

// Ausschalten macht die Kreuzung dunkel und haelt den Phasenautomaten an.
func TestSwitchingOffDarkensTheIntersection(t *testing.T) {
	h := newPowerHarness(t)
	h.run(8 * time.Second)
	if !h.controller.On() {
		t.Fatal("die Anlage gilt als ausgeschaltet")
	}

	h.flip(false)
	if h.controller.On() {
		t.Fatal("der Hauptschalter blieb ohne Wirkung")
	}
	if !allShow(aspectsOf(h.mock.Pattern()), light.AspectOff) {
		t.Errorf("nach dem Ausschalten zeigt die Kreuzung %v", aspectsOf(h.mock.Pattern()))
	}

	writes := h.mock.Writes()
	frozen := h.controller.State()
	h.run(2 * time.Minute)
	if got := h.mock.Writes(); got != writes {
		t.Errorf("%d Schreibzugriffe im ausgeschalteten Zustand", got-writes)
	}
	if got := h.controller.State(); got != frozen {
		t.Errorf("der Automat lief weiter von %s auf %s", frozen.Name(), got.Name())
	}
}

// Beim Einschalten beginnt die Anlage mit Allrot. Aus dem dunklen Zustand darf nie
// unmittelbar eine Freigabe folgen.
func TestSwitchingOnStartsFromAllRed(t *testing.T) {
	h := newPowerHarness(t)
	h.run(8 * time.Second)
	h.flip(false)

	mark := len(h.mock.History())
	h.flip(true)
	if !h.controller.On() {
		t.Fatal("die Anlage blieb ausgeschaltet")
	}

	first := aspectsOf(h.mock.History()[mark])
	if !allShow(first, light.AspectRed) {
		t.Errorf("das erste Bild nach dem Einschalten ist %v, erwartet Allrot", first)
	}
	h.run(2 * time.Minute)

	previous := aspectsOf(h.mock.History()[mark])
	for i, pattern := range h.mock.History()[mark+1:] {
		current := aspectsOf(pattern)
		if err := Check(current); err != nil {
			t.Fatalf("muster %d ist unzulaessig: %v", i, err)
		}
		for _, direction := range light.Directions() {
			if !current[direction].CanFollow(previous[direction]) {
				t.Fatalf("muster %d: %s wechselt von %s auf %s",
					i, direction, previous[direction], current[direction])
			}
		}
		previous = current
	}
}

// Das Einschalten beginnt eine neue Messung und damit einen neuen Lauf im Log.
func TestSwitchingOnStartsANewMeasurement(t *testing.T) {
	h := newPowerHarness(t)
	h.run(time.Second)
	for _, pin := range []int{12, 26, 19} {
		h.controller.Feed(Input{Pin: pin, Active: true, Time: h.clk.Now()})
	}
	h.run(time.Second)

	h.flip(false)
	h.flip(true)

	if got := h.controller.Snapshot(h.clk.Now()).Queues[light.East]; got != 0 {
		t.Errorf("Rueckstau Ost nach dem Einschalten %d, erwartet null", got)
	}
	if h.controller.Metrics().Total() != 0 {
		t.Errorf("%d Fahrzeuge in den Kennzahlen", h.controller.Metrics().Total())
	}
	if h.observer.power != 2 {
		t.Errorf("%d Schaltmarken im Log, erwartet zwei", h.observer.power)
	}
}

// Ein prellender Schalter darf die Anlage nicht abschalten.
func TestBouncingSwitchIsIgnored(t *testing.T) {
	h := newPowerHarness(t)
	h.run(2 * time.Second)
	for i := 0; i < 8; i++ {
		h.controller.Feed(Input{Pin: powerPin, Active: i%2 == 0, Time: h.clk.Now()})
		h.run(50 * time.Millisecond)
	}
	h.controller.Feed(Input{Pin: powerPin, Active: true, Time: h.clk.Now()})
	h.run(time.Second)

	if !h.controller.On() {
		t.Error("die Anlage schaltete sich durch Prellen ab")
	}
	if h.observer.power != 0 {
		t.Errorf("%d Schaltmarken durch Prellen", h.observer.power)
	}
}

// Aus dem Notzustand fuehrt der Hauptschalter heraus: aus und wieder an ist der Neustart der
// Anlage, den die Sicherheitsregel verlangt.
func TestPowerCycleLeavesTheFaultState(t *testing.T) {
	h := newPowerHarness(t)
	h.run(4 * time.Second)
	h.clk.Advance(DefaultWatchdog + 100*time.Millisecond)
	h.controller.Step(h.clk.Now())
	if h.controller.State().Phase != PhaseFault {
		t.Fatal("der Watchdog loeste nicht aus")
	}

	h.flip(false)
	h.flip(true)
	if got := h.controller.State().Phase; got == PhaseFault {
		t.Error("die Anlage blieb nach dem Aus- und Einschalten gestoert")
	}
	h.run(time.Minute)
	if got := h.controller.State(); got.Phase == PhaseFault {
		t.Errorf("die Anlage ging erneut in Stoerung: %v", h.observer.faults)
	}
}
