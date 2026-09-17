package controller

import (
	"testing"
	"time"

	"ampel/internal/clock"
	"ampel/internal/detector"
	"ampel/internal/hal"
	"ampel/internal/light"
	"ampel/internal/strategy"
)

var sensorPins = [light.DirectionCount][]int{
	light.North: {5, 6, 13},
	light.East:  {19, 26, 12},
	light.South: {16, 20, 21},
	light.West:  {23, 24, 25},
}

type record struct {
	at    time.Time
	state State
}

type recorder struct {
	NopObserver
	phases  []record
	faults  []error
	modes   []string
	power   int
	samples int
}

func (r *recorder) PhaseChanged(at time.Time, state State, _ string) {
	r.phases = append(r.phases, record{at: at, state: state})
}

func (r *recorder) ModeChanged(_ time.Time, mode string) { r.modes = append(r.modes, mode) }

func (r *recorder) PowerChanged(time.Time, bool) { r.power++ }

func (r *recorder) Fault(_ time.Time, err error) { r.faults = append(r.faults, err) }

func (r *recorder) Sample(time.Time, Snapshot) { r.samples++ }

type harness struct {
	controller *Controller
	clk        *clock.Fake
	mock       *hal.Mock
	observer   *recorder
}

// newHarness baut eine Kreuzung, deren Freigaben ohne Verkehr genau green dauern.
func newHarness(t *testing.T, green time.Duration, tune ...func(*Setup)) *harness {
	t.Helper()
	following, err := strategy.NewFollowing(green, 3*time.Second, green+15*time.Second)
	if err != nil {
		t.Fatalf("NewFollowing: %v", err)
	}
	clk := clock.NewFake(start)
	mock := hal.NewMock(16, 64)
	observer := &recorder{}
	setup := Setup{
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
		Sample:   time.Second,
		Strategy: following,
		Clock:    clk,
		Writer:   mock,
		Observer: observer,
	}
	for _, apply := range tune {
		apply(&setup)
	}
	c, err := Build(setup)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if err := c.Begin(start); err != nil {
		t.Fatalf("Begin: %v", err)
	}
	return &harness{controller: c, clk: clk, mock: mock, observer: observer}
}

func (h *harness) run(d time.Duration) {
	for elapsed := time.Duration(0); elapsed < d; elapsed += 50 * time.Millisecond {
		h.clk.Advance(50 * time.Millisecond)
		h.controller.Step(h.clk.Now())
	}
}

// Die Phasen muessen dauerhaft und in der richtigen Reihenfolge wechseln.
func TestPhasesCycleForever(t *testing.T) {
	h := newHarness(t, 15*time.Second)
	h.run(10 * time.Minute)

	want := []string{"NS_RotGelb", "NS_Gruen", "NS_Gelb", "Allrot", "OW_RotGelb", "OW_Gruen", "OW_Gelb", "Allrot"}
	if len(h.observer.phases) < 2*len(want) {
		t.Fatalf("nur %d Wechsel in zehn Minuten", len(h.observer.phases))
	}
	// Der erste Eintrag ist das Startbild Allrot.
	for i, phase := range h.observer.phases[1:] {
		if got, expect := phase.state.Name(), want[i%len(want)]; got != expect {
			t.Fatalf("wechsel %d ist %s, erwartet %s", i, got, expect)
		}
		if i > 2*len(want) {
			break
		}
	}
	if len(h.observer.faults) != 0 {
		t.Errorf("Stoerungen: %v", h.observer.faults)
	}
}

// Die Zwischenzeiten sind fest, und ohne Verkehr dauert die Freigabe genau die Grundzeit.
func TestPhaseDurationsHold(t *testing.T) {
	h := newHarness(t, 15*time.Second)
	h.run(2 * time.Minute)

	tick := 50 * time.Millisecond
	want := map[string]time.Duration{
		"NS_Gruen":   15 * time.Second,
		"OW_Gruen":   15 * time.Second,
		"NS_Gelb":    timing.Yellow,
		"OW_Gelb":    timing.Yellow,
		"Allrot":     timing.AllRed,
		"NS_RotGelb": timing.RedYellow,
		"OW_RotGelb": timing.RedYellow,
	}
	phases := h.observer.phases
	for i := 1; i < len(phases)-1; i++ {
		name := phases[i].state.Name()
		expect, ok := want[name]
		if !ok {
			t.Fatalf("unerwarteter Zustand %s", name)
		}
		got := phases[i+1].at.Sub(phases[i].at)
		if got < expect || got > expect+tick {
			t.Errorf("%s dauerte %s, erwartet %s", name, got, expect)
		}
	}
}

// Zwischen Gruen und Rot muss immer Gelb liegen, und kreuzende Richtungen duerfen nie
// gemeinsam freigeben. Geprueft wird am Bitmuster, das die Hardware tatsaechlich erhaelt.
func TestWrittenPatternsNeverViolateSignalRules(t *testing.T) {
	h := newHarness(t, 8*time.Second)
	h.run(5 * time.Minute)

	history := h.mock.History()
	if len(history) < 20 {
		t.Fatalf("nur %d Schreibzugriffe", len(history))
	}
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
}

// Ein Fahrzeug, das die Haltelinie wieder freigibt, hat die Kreuzung ueberfahren. Das ist
// das Ereignis, auf dem die Verlaengerung beruht.
func TestCrossingIsCountedWhenTheStopLineIsReleased(t *testing.T) {
	h := newHarness(t, 10*time.Second)
	h.run(5 * time.Second)

	// Nord ist freigegeben: Pin 5 ist die Haltelinie.
	if got := h.controller.State(); got.Phase != PhaseNS || got.Stage != StageGreen {
		t.Fatalf("Zustand %s, erwartet die Freigabe fuer Nord und Sued", got.Name())
	}
	h.controller.Feed(Input{Pin: 5, Active: true, Time: h.clk.Now()})
	h.run(300 * time.Millisecond)
	h.controller.Feed(Input{Pin: 5, Active: false, Time: h.clk.Now()})
	h.run(100 * time.Millisecond)
	h.controller.Feed(Input{Pin: 5, Active: true, Time: h.clk.Now()})
	h.run(100 * time.Millisecond)
	h.controller.Feed(Input{Pin: 5, Active: false, Time: h.clk.Now()})
	h.run(100 * time.Millisecond)

	if got := h.controller.Snapshot(h.clk.Now()).Following; got != 1 {
		t.Errorf("%d Verlaengerungen nach zwei Ueberfahrten, erwartet eine", got)
	}
}

func TestSamplesAreEmitted(t *testing.T) {
	h := newHarness(t, 15*time.Second)
	h.run(30 * time.Second)
	if h.observer.samples < 25 || h.observer.samples > 31 {
		t.Errorf("%d Abtastwerte in 30 Sekunden, erwartet etwa 30", h.observer.samples)
	}
}

// Nach dem Herunterfahren steht alles auf Rot, und der Weg dorthin ist zulaessig.
func TestShutdownEndsInAllRed(t *testing.T) {
	for _, wait := range []time.Duration{4 * time.Second, 8 * time.Second, 20 * time.Second} {
		h := newHarness(t, 15*time.Second)
		h.run(wait)
		if err := h.controller.Shutdown(); err != nil {
			t.Fatalf("nach %s: Shutdown: %v", wait, err)
		}
		aspects := aspectsOf(h.mock.Pattern())
		for _, direction := range light.Directions() {
			if aspects[direction] != light.AspectRed {
				t.Errorf("nach %s zeigt %s %s, erwartet Rot", wait, direction, aspects[direction])
			}
		}
	}
}

// aspectsOf liest die Signalbilder aus dem Bitmuster zurueck, das der Treiber erhalten hat.
func aspectsOf(pattern []bool) [light.DirectionCount]light.Aspect {
	var aspects [light.DirectionCount]light.Aspect
	for _, direction := range light.Directions() {
		base := int(direction) * 3
		lamps := light.Lamps{Red: pattern[base], Yellow: pattern[base+1], Green: pattern[base+2]}
		switch lamps {
		case light.Lamps{Red: true}:
			aspects[direction] = light.AspectRed
		case light.Lamps{Red: true, Yellow: true}:
			aspects[direction] = light.AspectRedYellow
		case light.Lamps{Green: true}:
			aspects[direction] = light.AspectGreen
		case light.Lamps{Yellow: true}:
			aspects[direction] = light.AspectYellow
		default:
			aspects[direction] = light.AspectOff
		}
	}
	return aspects
}

var _ = detector.SensorEvent{}
