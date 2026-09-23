package controller

import (
	"testing"
	"time"

	"ampel/internal/light"
)

const (
	powerPin = 4
	faultPin = 18
)

// newPowerHarness startet mit eingeschalteter Anlage.
func newPowerHarness(t *testing.T) *harness {
	t.Helper()
	return newHarness(t, 15*time.Second, func(setup *Setup) {
		setup.Switches = &Switches{PowerPin: powerPin, FaultPin: faultPin, PowerOn: true}
	})
}

func (h *harness) flip(pin int, on bool) {
	h.controller.Feed(Input{Pin: pin, Active: on, Time: h.clk.Now()})
	h.run(300 * time.Millisecond)
}

// Ausschalten macht die Kreuzung dunkel und haelt den Phasenautomaten an.
func TestSwitchingOffDarkensTheIntersection(t *testing.T) {
	h := newPowerHarness(t)
	h.run(8 * time.Second)
	if !h.controller.On() {
		t.Fatal("die Anlage gilt als ausgeschaltet")
	}

	h.flip(powerPin, false)
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
	h.flip(powerPin, false)

	mark := len(h.mock.History())
	h.flip(powerPin, true)
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
	h.controller.Feed(Input{Pin: 24, Active: true, Time: h.clk.Now()})
	h.run(time.Second)

	h.flip(powerPin, false)
	h.flip(powerPin, true)

	if got := h.controller.Snapshot(h.clk.Now()).Following; got != 0 {
		t.Errorf("%d Verlaengerungen nach dem Einschalten", got)
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

	h.flip(powerPin, false)
	h.flip(powerPin, true)
	if got := h.controller.State().Phase; got == PhaseFault {
		t.Error("die Anlage blieb nach dem Aus- und Einschalten gestoert")
	}
	h.run(time.Minute)
	if got := h.controller.State(); got.Phase == PhaseFault {
		t.Errorf("die Anlage ging erneut in Stoerung: %v", h.observer.faults)
	}
}

// Der Notschalter laesst alle Lichter gelb blinken und haelt den Automaten an. Zurueckgelegt
// beginnt die Anlage wieder bei Allrot.
func TestFaultSwitchBlinksAndRestarts(t *testing.T) {
	h := newPowerHarness(t)
	h.run(8 * time.Second)

	h.flip(faultPin, true)
	mark := len(h.mock.History())
	h.run(10 * time.Second)

	pulses := 0
	for _, pattern := range h.mock.History()[mark:] {
		switch aspects := aspectsOf(pattern); {
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

	h.flip(faultPin, false)
	first := aspectsOf(h.mock.Pattern())
	if !allShow(first, light.AspectRed) {
		t.Errorf("nach dem Zuruecklegen zeigt die Kreuzung %v, erwartet Allrot", first)
	}
	h.run(time.Minute)
	if got := h.controller.State(); got.Phase == PhaseFault {
		t.Error("die Anlage blieb im Notzustand")
	}
}

// Im Notzustand leuchtet nur die mittlere Lampe. Rot und Gruen bleiben dunkel, wie bei einer
// abgeschalteten Anlage im Strassenverkehr.
func TestWarningUsesOnlyTheYellowLamp(t *testing.T) {
	h := newPowerHarness(t)
	h.run(8 * time.Second)
	h.flip(faultPin, true)

	mark := len(h.mock.History())
	h.run(3 * time.Second)

	lit := false
	for _, pattern := range h.mock.History()[mark:] {
		for _, direction := range light.Directions() {
			base := int(direction) * 3
			if pattern[base] || pattern[base+2] {
				t.Fatalf("%s zeigt Rot oder Gruen im Notzustand: %v", direction, pattern[base:base+3])
			}
			if pattern[base+1] {
				lit = true
			}
		}
	}
	if !lit {
		t.Error("die Gelblampe leuchtete kein einziges Mal")
	}
}
