package controller

import (
	"testing"
	"time"

	"ampel/internal/light"
	"ampel/internal/strategy"
)

const (
	switchPin = 4
	resetPin  = 18
)

var params = strategy.Params{
	MinGreen:   5 * time.Second,
	MaxGreen:   25 * time.Second,
	Cycle:      40 * time.Second,
	Intergreen: timing.Intergreen(),
	Gap:        2 * time.Second,
	Extension:  1500 * time.Millisecond,
	MaxWait:    60 * time.Second,
}

// spyLearner zaehlt, wie oft der Lernzustand geloescht und beobachtet wurde.
type spyLearner struct {
	NopLearner
	observed int
	resets   int
}

func (s *spyLearner) Observe(time.Time, light.Direction, float64) { s.observed++ }

func (s *spyLearner) Reset() { s.resets++ }

// newPanelHarness startet im Festzeitbetrieb mit offenem Kippschalter.
func newPanelHarness(t *testing.T, learner *spyLearner) *harness {
	t.Helper()
	adaptive, err := strategy.NewAdaptive(params)
	if err != nil {
		t.Fatalf("NewAdaptive: %v", err)
	}
	fixed := strategy.NewFixed(15 * time.Second)
	return newHarness(t, 15*time.Second, func(setup *Setup) {
		setup.Strategy = fixed
		setup.Learner = learner
		setup.Panel = &Panel{
			SwitchPin: switchPin,
			ResetPin:  resetPin,
			Fixed:     fixed,
			Adaptive:  adaptive,
		}
	})
}

func (h *harness) press(pin int, active bool) {
	h.controller.Feed(Input{Pin: pin, Active: active, Time: h.clk.Now()})
}

// waitForMode laeuft, bis sich die Betriebsart aendert, und liefert den Zustand in diesem
// Augenblick.
func (h *harness) waitForMode(from string, limit time.Duration) (State, time.Time, bool) {
	for elapsed := time.Duration(0); elapsed < limit; elapsed += 50 * time.Millisecond {
		h.clk.Advance(50 * time.Millisecond)
		h.controller.Step(h.clk.Now())
		if h.controller.Mode() != from {
			return h.controller.State(), h.clk.Now(), true
		}
	}
	return State{}, time.Time{}, false
}

// Der Kippschalter wirkt nicht mitten in der Freigabe, sondern erst beim naechsten
// Phasenwechsel.
func TestModeSwitchTakesEffectAtGreenStart(t *testing.T) {
	h := newPanelHarness(t, &spyLearner{})
	h.run(5 * time.Second)
	if got := h.controller.Mode(); got != "festzeit" {
		t.Fatalf("Startmodus %s, erwartet festzeit", got)
	}
	before := h.controller.State()
	if before.Stage != StageGreen {
		t.Fatalf("Zustand %s, erwartet eine laufende Freigabe", before.Name())
	}

	h.press(switchPin, true)
	state, at, ok := h.waitForMode("festzeit", 2*time.Minute)
	if !ok {
		t.Fatal("der Kippschalter blieb ohne Wirkung")
	}
	if h.controller.Mode() != "adaptiv" {
		t.Fatalf("Modus %s, erwartet adaptiv", h.controller.Mode())
	}
	if state.Stage != StageGreen || !state.Since.Equal(at) {
		t.Errorf("Wechsel im Zustand %s nach %s, erwartet den Beginn einer Freigabe",
			state.Name(), at.Sub(state.Since))
	}
	if state.Phase == before.Phase && at.Sub(before.Since) < timing.Intergreen() {
		t.Error("der Wechsel wirkte noch in derselben Freigabe")
	}
	if len(h.observer.modes) != 1 || h.observer.modes[0] != "adaptiv" {
		t.Errorf("gemeldete Moduswechsel %v, erwartet genau einen nach adaptiv", h.observer.modes)
	}
}

// Ein prellender Kippschalter darf keinen Moduswechsel ausloesen.
func TestModeSwitchIgnoresBouncing(t *testing.T) {
	h := newPanelHarness(t, &spyLearner{})
	h.run(2 * time.Second)
	for i := 0; i < 8; i++ {
		h.press(switchPin, i%2 == 0)
		h.run(50 * time.Millisecond)
	}
	h.press(switchPin, false)
	h.run(time.Minute)

	if got := h.controller.Mode(); got != "festzeit" {
		t.Errorf("Modus %s, erwartet festzeit", got)
	}
	if len(h.observer.modes) != 0 {
		t.Errorf("gemeldete Moduswechsel %v, erwartet keinen", h.observer.modes)
	}
}

// Das Umschalten im Betrieb darf nie ein unzulaessiges Signalbild erzeugen. Der Schalter
// kippt dabei absichtlich in jeder Phasenlage.
func TestSwitchingNeverBreaksSignalRules(t *testing.T) {
	h := newPanelHarness(t, &spyLearner{})
	closed := false
	for i := 0; i < 230; i++ {
		closed = !closed
		h.press(switchPin, closed)
		h.run(1300 * time.Millisecond)
	}

	history := h.mock.History()
	previous := aspectsOf(history[0])
	for i, pattern := range history[1:] {
		current := aspectsOf(pattern)
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
	if len(h.observer.faults) != 0 {
		t.Errorf("Stoerungen: %v", h.observer.faults)
	}
	if len(h.observer.modes) < 4 {
		t.Errorf("nur %d Moduswechsel, der Schalter blieb wirkungslos", len(h.observer.modes))
	}
}

// Der Festzeitbetrieb misst mit, lernt aber nicht. Erst der adaptive Betrieb fuellt das
// Tagesprofil.
func TestFixedModeDoesNotLearn(t *testing.T) {
	learner := &spyLearner{}
	h := newPanelHarness(t, learner)
	h.run(2 * time.Minute)
	if learner.observed != 0 {
		t.Fatalf("%d Beobachtungen im Festzeitbetrieb, erwartet keine", learner.observed)
	}

	h.press(switchPin, true)
	h.run(2 * time.Minute)
	if learner.observed == 0 {
		t.Error("der adaptive Betrieb lernte nichts")
	}
}
func allShow(aspects [light.DirectionCount]light.Aspect, want light.Aspect) bool {
	for _, aspect := range aspects {
		if aspect != want {
			return false
		}
	}
	return true
}
