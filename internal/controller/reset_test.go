package controller

import (
	"testing"
	"time"

	"ampel/internal/light"
)

// Der Reset loest erst nach zwei Sekunden Dauerdruck aus und quittiert mit dreimaligem
// Blinken aller Gelblichter.
func TestResetNeedsLongPressAndBlinks(t *testing.T) {
	h := newPanelHarness(t)
	h.run(4 * time.Second)

	h.press(resetPin, true)
	h.run(time.Second)
	h.press(resetPin, false)
	h.run(time.Second)
	if h.observer.resets != 0 {
		t.Fatal("ein kurzer Druck loeste den Reset aus")
	}

	h.press(resetPin, true)
	h.run(2100 * time.Millisecond)
	if h.observer.resets != 1 {
		t.Fatalf("%d Resetmarken nach langem Druck, erwartet eine", h.observer.resets)
	}
	h.press(resetPin, false)

	mark := len(h.mock.History())
	// Die Quittung wartet Gelb und Allrot ab, blinkt dreimal und gibt danach wieder frei.
	h.run(20 * time.Second)

	yellow, dark := 0, 0
	for _, pattern := range h.mock.History()[mark:] {
		switch aspects := aspectsOf(pattern); {
		case allShow(aspects, light.AspectYellow):
			yellow++
		case allShow(aspects, light.AspectOff):
			dark++
		}
	}
	if yellow != 3 || dark != 3 {
		t.Errorf("%d Gelbimpulse und %d Dunkelphasen, erwartet je drei", yellow, dark)
	}
	if h.controller.State().Phase == PhaseFault {
		t.Fatal("die Quittung fuehrte in den Notzustand")
	}
	if len(h.observer.faults) != 0 {
		t.Errorf("Stoerungen: %v", h.observer.faults)
	}
}

// Nach der Quittung nimmt die Kreuzung den Betrieb wieder auf, ohne die Signalfolge zu
// verletzen.
func TestResumesAfterAcknowledgement(t *testing.T) {
	h := newPanelHarness(t)
	h.run(4 * time.Second)
	h.press(resetPin, true)
	h.run(2100 * time.Millisecond)
	h.press(resetPin, false)
	h.run(2 * time.Minute)

	history := h.mock.History()
	previous := aspectsOf(history[0])
	for i, pattern := range history[1:] {
		current := aspectsOf(pattern)
		// Die Blinkquittung ist kein Signalbild. Aus dem Bitmuster ist sie nicht vom
		// stehenden Gelb zu unterscheiden, deshalb wird sie hier uebersprungen.
		if allShow(current, light.AspectYellow) || allShow(current, light.AspectOff) {
			previous = current
			continue
		}
		if err := Check(current); err != nil {
			t.Fatalf("muster %d ist unzulaessig: %v", i+1, err)
		}
		for _, direction := range light.Directions() {
			if !current[direction].CanFollow(previous[direction]) {
				t.Fatalf("muster %d: %s wechselt von %s auf %s",
					i+1, direction, previous[direction], current[direction])
			}
		}
		previous = current
	}
	greens := 0
	for _, phase := range h.observer.phases {
		if phase.state.Stage == StageGreen && phase.at.After(start.Add(10*time.Second)) {
			greens++
		}
	}
	if greens < 3 {
		t.Errorf("nur %d Freigaben nach der Quittung", greens)
	}
}
