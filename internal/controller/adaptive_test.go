package controller

import (
	"testing"
	"time"

	"ampel/internal/clock"
	"ampel/internal/hal"
	"ampel/internal/light"
	"ampel/internal/strategy"
)

// load erzeugt Verkehr auf einer Zufahrt: ein Dauerstau hinter der Haltelinie und ein
// Fahrzeug, das im Abstand interval ueber die Linie faehrt, solange freigegeben ist.
type load struct {
	direction light.Direction
	stopLine  int
	upstream  []int
	interval  time.Duration
	next      time.Duration
}

func newAdaptiveHarness(t *testing.T) *harness {
	t.Helper()
	clk := clock.NewFake(start)
	mock := hal.NewMock(16, 4096)
	observer := &recorder{}
	adaptive, err := strategy.NewFollowing(8*time.Second, 3*time.Second, 30*time.Second)
	if err != nil {
		t.Fatalf("NewFollowing: %v", err)
	}
	c, err := Build(Setup{
		Sensors:  sensorPins,
		Debounce: 15 * time.Millisecond,
		LampMatrix: [light.DirectionCount][3]int{
			light.North: {0, 1, 2},
			light.East:  {3, 4, 5},
			light.South: {6, 7, 8},
			light.West:  {9, 10, 11},
		},
		Bits:     16,
		Timing:   timing,
		Tick:     50 * time.Millisecond,
		Strategy: adaptive,
		Clock:    clk,
		Writer:   mock,
		Observer: observer,
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if err := c.Begin(start); err != nil {
		t.Fatalf("Begin: %v", err)
	}
	return &harness{controller: c, clk: clk, mock: mock, observer: observer}
}

// Dicht aufeinander folgende Fahrzeuge muessen die Freigabe ihrer Richtung verlaengern, eine
// Richtung mit vereinzeltem Verkehr behaelt die Grundzeit.
func TestAdaptiveFavoursLoadedDirection(t *testing.T) {
	h := newAdaptiveHarness(t)
	loads := []*load{
		{direction: light.North, stopLine: 5, upstream: []int{6, 13}, interval: 700 * time.Millisecond},
		{direction: light.South, stopLine: 16, upstream: []int{20, 21}, interval: 700 * time.Millisecond},
		{direction: light.East, stopLine: 19, upstream: []int{26}, interval: 6 * time.Second},
		{direction: light.West, stopLine: 23, upstream: []int{24}, interval: 6 * time.Second},
	}
	h.drive(loads, 15*time.Minute)

	ns := h.meanGreen("NS_Gruen")
	ew := h.meanGreen("OW_Gruen")
	if ns == 0 || ew == 0 {
		t.Fatalf("Gruenzeiten fehlen: NS %s, OW %s", ns, ew)
	}
	if ns < ew+5*time.Second {
		t.Errorf("Nord und Sued erhalten %s Gruen, Ost und West %s; erwartet mindestens fuenf Sekunden mehr", ns, ew)
	}
	if ns > 30*time.Second+time.Second {
		t.Errorf("Gruenzeit %s ueberschreitet die Hoechstgruenzeit", ns)
	}
	if len(h.observer.faults) != 0 {
		t.Errorf("Stoerungen: %v", h.observer.faults)
	}
}

// Gegenprobe: liegen die Fahrzeuge weiter auseinander als die Folgezeit, bleibt es bei der
// Grundzeit, egal wie viele kommen.
func TestScatteredLoadKeepsBaseGreen(t *testing.T) {
	h := newAdaptiveHarness(t)
	loads := []*load{
		{direction: light.North, stopLine: 5, upstream: []int{6, 13}, interval: 5 * time.Second},
		{direction: light.South, stopLine: 16, upstream: []int{20, 21}, interval: 5 * time.Second},
		{direction: light.East, stopLine: 19, upstream: []int{26}, interval: 5 * time.Second},
		{direction: light.West, stopLine: 23, upstream: []int{24}, interval: 5 * time.Second},
	}
	h.drive(loads, 5*time.Minute)

	ns, ew := h.meanGreen("NS_Gruen"), h.meanGreen("OW_Gruen")
	for name, green := range map[string]time.Duration{"NS": ns, "OW": ew} {
		if green > 9*time.Second {
			t.Errorf("%s bekommt %s Gruen, erwartet die Grundzeit von 8s", name, green)
		}
	}
}

// drive laesst den Regelkreis laufen und speist dabei den Verkehr ein.
func (h *harness) drive(loads []*load, duration time.Duration) {
	const step = 50 * time.Millisecond
	for _, l := range loads {
		for _, pin := range l.upstream {
			h.controller.Feed(Input{Pin: pin, Active: true, Time: h.clk.Now()})
		}
		h.controller.Feed(Input{Pin: l.stopLine, Active: true, Time: h.clk.Now()})
	}

	for elapsed := time.Duration(0); elapsed < duration; elapsed += step {
		h.clk.Advance(step)
		now := h.clk.Now()
		h.controller.Step(now)

		state := h.controller.State()
		for _, l := range loads {
			if state.Stage != StageGreen || !releases(state.Phase, l.direction) {
				continue
			}
			if elapsed < l.next {
				continue
			}
			l.next = elapsed + l.interval
			// Ein Fahrzeug faehrt ueber die Linie, das naechste rollt nach.
			h.controller.Feed(Input{Pin: l.stopLine, Active: false, Time: now})
			h.clk.Advance(step)
			h.controller.Step(h.clk.Now())
			h.controller.Feed(Input{Pin: l.stopLine, Active: true, Time: h.clk.Now()})
			elapsed += step
		}
	}
}

// meanGreen ist die mittlere Dauer aller vollstaendigen Abschnitte mit diesem Namen.
func (h *harness) meanGreen(name string) time.Duration {
	var sum time.Duration
	var count int
	phases := h.observer.phases
	for i := 0; i < len(phases)-1; i++ {
		if phases[i].state.Name() != name {
			continue
		}
		sum += phases[i+1].at.Sub(phases[i].at)
		count++
	}
	if count == 0 {
		return 0
	}
	return sum / time.Duration(count)
}

func releases(phase Phase, direction light.Direction) bool {
	for _, released := range phase.Directions() {
		if released == direction {
			return true
		}
	}
	return false
}
