package steuerung

import (
	"ampel/src/erkennung"
	"ampel/src/regel"
	"ampel/src/signal"
	"ampel/src/treiber/mock"
	"ampel/src/zeit"
	"errors"
	"testing"
	"time"
)

const (
	powerPin = 4
	faultPin = 18
)

var sensorPins = [signal.DirectionCount]int{
	signal.North: 23,
	signal.East:  24,
	signal.South: 25,
	signal.West:  8,
}

type record struct {
	at    time.Time
	state State
}

type recorder struct {
	NopObserver
	phases  []record
	faults  []error
	power   int
	samples int
}

func (r *recorder) PhaseChanged(at time.Time, state State) {
	r.phases = append(r.phases, record{at: at, state: state})
}

func (r *recorder) PowerChanged(time.Time, bool) { r.power++ }

func (r *recorder) Fault(_ time.Time, err error) { r.faults = append(r.faults, err) }

func (r *recorder) Sample(time.Time, Snapshot) { r.samples++ }

type harness struct {
	controller *Controller
	clk        *zeit.Fake
	mock       *mock.Mock
	observer   *recorder
}

func newHarness(t *testing.T, green time.Duration, tune ...func(*Setup)) *harness {
	t.Helper()
	following, err := regel.NewFollowing(green, 3*time.Second, green+15*time.Second)
	if err != nil {
		t.Fatalf("NewFollowing: %v", err)
	}
	clk := zeit.NewFake(start)
	mock := mock.NewMock(LampCount, 64)
	observer := &recorder{}
	setup := Setup{
		Sensors:  sensorPins,
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

func TestPhasesCycleForever(t *testing.T) {
	h := newHarness(t, 15*time.Second)
	h.run(10 * time.Minute)

	want := []string{"NS_RotGelb", "NS_Gruen", "NS_Gelb", "Allrot", "OW_RotGelb", "OW_Gruen", "OW_Gelb", "Allrot"}
	if len(h.observer.phases) < 2*len(want) {
		t.Fatalf("nur %d Wechsel in zehn Minuten", len(h.observer.phases))
	}
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
		for _, direction := range signal.Directions() {
			if !current[direction].CanFollow(previous[direction]) {
				t.Fatalf("muster %d: %s wechselt von %s auf %s",
					i+1, direction, previous[direction], current[direction])
			}
		}
		previous = current
	}
}

func TestCrossingIsCountedWhenTheStopLineIsReleased(t *testing.T) {
	h := newHarness(t, 10*time.Second)
	h.run(5 * time.Second)

	if got := h.controller.State(); got.Phase != PhaseNS || got.Stage != StageGreen {
		t.Fatalf("Zustand %s, erwartet die Freigabe fuer Nord und Sued", got.Name())
	}
	h.controller.Feed(Input{Pin: 23, Active: true, Time: h.clk.Now()})
	h.run(300 * time.Millisecond)
	h.controller.Feed(Input{Pin: 23, Active: false, Time: h.clk.Now()})
	h.run(100 * time.Millisecond)
	h.controller.Feed(Input{Pin: 23, Active: true, Time: h.clk.Now()})
	h.run(100 * time.Millisecond)
	h.controller.Feed(Input{Pin: 23, Active: false, Time: h.clk.Now()})
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

func TestShutdownEndsInAllRed(t *testing.T) {
	for _, wait := range []time.Duration{4 * time.Second, 8 * time.Second, 20 * time.Second} {
		h := newHarness(t, 15*time.Second)
		h.run(wait)
		if err := h.controller.Shutdown(); err != nil {
			t.Fatalf("nach %s: Shutdown: %v", wait, err)
		}
		aspects := aspectsOf(h.mock.Pattern())
		for _, direction := range signal.Directions() {
			if aspects[direction] != signal.AspectRed {
				t.Errorf("nach %s zeigt %s %s, erwartet Rot", wait, direction, aspects[direction])
			}
		}
	}
}

func aspectsOf(pattern []bool) [signal.DirectionCount]signal.Aspect {
	var aspects [signal.DirectionCount]signal.Aspect
	for _, direction := range signal.Directions() {
		base := int(direction) * 3
		lamps := signal.Lamps{Red: pattern[base], Yellow: pattern[base+1], Green: pattern[base+2]}
		switch lamps {
		case signal.Lamps{Red: true}:
			aspects[direction] = signal.AspectRed
		case signal.Lamps{Red: true, Yellow: true}:
			aspects[direction] = signal.AspectRedYellow
		case signal.Lamps{Green: true}:
			aspects[direction] = signal.AspectGreen
		case signal.Lamps{Yellow: true}:
			aspects[direction] = signal.AspectYellow
		default:
			aspects[direction] = signal.AspectOff
		}
	}
	return aspects
}

var _ = erkennung.SensorEvent{}

type load struct {
	direction signal.Direction
	stopLine  int
	interval  time.Duration
	next      time.Duration
}

func newAdaptiveHarness(t *testing.T) *harness {
	t.Helper()
	clk := zeit.NewFake(start)
	mock := mock.NewMock(LampCount, 4096)
	observer := &recorder{}
	adaptive, err := regel.NewFollowing(8*time.Second, 3*time.Second, 30*time.Second)
	if err != nil {
		t.Fatalf("NewFollowing: %v", err)
	}
	c, err := Build(Setup{
		Sensors:  sensorPins,
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

func TestAdaptiveFavoursLoadedDirection(t *testing.T) {
	h := newAdaptiveHarness(t)
	loads := []*load{
		{direction: signal.North, stopLine: 23, interval: 700 * time.Millisecond},
		{direction: signal.South, stopLine: 25, interval: 700 * time.Millisecond},
		{direction: signal.East, stopLine: 24, interval: 6 * time.Second},
		{direction: signal.West, stopLine: 8, interval: 6 * time.Second},
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

func TestScatteredLoadKeepsBaseGreen(t *testing.T) {
	h := newAdaptiveHarness(t)
	loads := []*load{
		{direction: signal.North, stopLine: 23, interval: 5 * time.Second},
		{direction: signal.South, stopLine: 25, interval: 5 * time.Second},
		{direction: signal.East, stopLine: 24, interval: 5 * time.Second},
		{direction: signal.West, stopLine: 8, interval: 5 * time.Second},
	}
	h.drive(loads, 5*time.Minute)

	ns, ew := h.meanGreen("NS_Gruen"), h.meanGreen("OW_Gruen")
	for name, green := range map[string]time.Duration{"NS": ns, "OW": ew} {
		if green > 9*time.Second {
			t.Errorf("%s bekommt %s Gruen, erwartet die Grundzeit von 8s", name, green)
		}
	}
}

func (h *harness) drive(loads []*load, duration time.Duration) {
	const step = 50 * time.Millisecond
	for _, l := range loads {
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
			h.controller.Feed(Input{Pin: l.stopLine, Active: false, Time: now})
			h.clk.Advance(step)
			h.controller.Step(h.clk.Now())
			h.controller.Feed(Input{Pin: l.stopLine, Active: true, Time: h.clk.Now()})
			elapsed += step
		}
	}
}

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

func releases(phase Phase, direction signal.Direction) bool {
	for _, released := range phase.Directions() {
		if released == direction {
			return true
		}
	}
	return false
}

func TestDisplayedGreenJumpsWhenVehiclesFollow(t *testing.T) {
	h := newAdaptiveHarness(t)
	h.run(7 * time.Second)

	state := h.controller.State()
	if state.Phase != PhaseNS || state.Stage != StageGreen {
		t.Fatalf("Zustand %s, erwartet eine laufende Freigabe fuer Nord und Sued", state.Name())
	}
	before := h.controller.Snapshot(h.clk.Now())

	loads := []*load{{direction: signal.North, stopLine: 23, interval: 700 * time.Millisecond}}
	h.drive(loads, 2*time.Second)

	after := h.controller.Snapshot(h.clk.Now())
	if after.Following <= before.Following {
		t.Fatalf("%d Verlaengerungen, vorher %d", after.Following, before.Following)
	}
	if after.Green[signal.North] <= before.Green[signal.North] {
		t.Errorf("Restzeit fiel von %s auf %s, erwartet einen Sprung nach oben",
			before.Green[signal.North], after.Green[signal.North])
	}
	if after.Green[signal.North] != after.Green[signal.South] {
		t.Errorf("Nord zeigt %s, Sued %s, beide teilen sich die Freigabe",
			after.Green[signal.North], after.Green[signal.South])
	}
	if got := after.Green[signal.East]; got != 8*time.Second {
		t.Errorf("die wartende Richtung zeigt %s, erwartet ihre Grundzeit von 8s", got)
	}
}

func TestScatteredTrafficDoesNotExtend(t *testing.T) {
	h := newAdaptiveHarness(t)
	loads := []*load{
		{direction: signal.North, stopLine: 23, interval: 5 * time.Second},
		{direction: signal.South, stopLine: 25, interval: 5 * time.Second},
	}
	h.drive(loads, 8*time.Second)

	if got := h.controller.Snapshot(h.clk.Now()).Following; got != 0 {
		t.Errorf("%d Verlaengerungen bei vereinzeltem Verkehr, erwartet keine", got)
	}
}

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

func TestSnapshotCarriesAspects(t *testing.T) {
	h := newAdaptiveHarness(t)
	h.run(12 * time.Second)
	snapshot := h.controller.Snapshot(h.clk.Now())
	if snapshot.Aspects != h.controller.State().Aspects() {
		t.Errorf("Signalbilder %v weichen vom Zustand ab", snapshot.Aspects)
	}
}

type counter struct {
	NopObserver
	calls int
}

func (c *counter) PhaseChanged(time.Time, State) { c.calls++ }
func (c *counter) PowerChanged(time.Time, bool)  { c.calls++ }
func (c *counter) Fault(time.Time, error)        { c.calls++ }
func (c *counter) Sample(time.Time, Snapshot)    { c.calls++ }

func newOutput(t *testing.T) (*Output, *mock.Mock) {
	t.Helper()
	mock := mock.NewMock(LampCount, 1)
	return NewOutput(mock), mock
}

func TestShowRejectsConflictBeforeWriting(t *testing.T) {
	output, mock := newOutput(t)
	if err := output.Show([signal.DirectionCount]signal.Aspect{
		signal.AspectRed, signal.AspectRed, signal.AspectRed, signal.AspectRed}); err != nil {
		t.Fatalf("alles auf Rot: %v", err)
	}
	writes := mock.Writes()

	err := output.Show([signal.DirectionCount]signal.Aspect{
		signal.AspectRedYellow, signal.AspectRedYellow, signal.AspectRed, signal.AspectRed})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("Fehler %v, erwartet ErrConflict", err)
	}
	if mock.Writes() != writes {
		t.Error("der Konflikt wurde in die Hardware geschrieben")
	}
	for i, aspect := range output.Aspects() {
		if aspect != signal.AspectRed {
			t.Errorf("Zufahrt %s zeigt %s, erwartet unveraendert Rot", signal.Direction(i), aspect)
		}
	}
}

func TestShowRejectsSequenceViolation(t *testing.T) {
	output, mock := newOutput(t)
	if err := output.Show([signal.DirectionCount]signal.Aspect{
		signal.AspectRed, signal.AspectRed, signal.AspectRed, signal.AspectRed}); err != nil {
		t.Fatalf("alles auf Rot: %v", err)
	}
	writes := mock.Writes()

	err := output.Show([signal.DirectionCount]signal.Aspect{
		signal.AspectGreen, signal.AspectRed, signal.AspectGreen, signal.AspectRed})
	if err == nil {
		t.Fatal("Gruen direkt nach Rot wurde angenommen")
	}
	if mock.Writes() != writes {
		t.Error("der unzulaessige Wechsel wurde in die Hardware geschrieben")
	}
}

func TestShowWritesPattern(t *testing.T) {
	output, mock := newOutput(t)
	steps := [][signal.DirectionCount]signal.Aspect{
		{signal.AspectRed, signal.AspectRed, signal.AspectRed, signal.AspectRed},
		{signal.AspectRedYellow, signal.AspectRed, signal.AspectRedYellow, signal.AspectRed},
		{signal.AspectGreen, signal.AspectRed, signal.AspectGreen, signal.AspectRed},
	}
	for _, step := range steps {
		if err := output.Show(step); err != nil {
			t.Fatalf("Show: %v", err)
		}
	}

	pattern := mock.Pattern()
	for _, bit := range []int{2, 8, 3, 9} {
		if !pattern[bit] {
			t.Errorf("bit %d leuchtet nicht", bit)
		}
	}
	if mock.Writes() != len(steps) {
		t.Errorf("%d Schreibzugriffe, erwartet %d", mock.Writes(), len(steps))
	}
}

func TestDarkTurnsEverythingOff(t *testing.T) {
	output, mock := newOutput(t)
	if err := output.Show([signal.DirectionCount]signal.Aspect{
		signal.AspectRed, signal.AspectRed, signal.AspectRed, signal.AspectRed}); err != nil {
		t.Fatalf("alles auf Rot: %v", err)
	}
	if err := output.Dark(); err != nil {
		t.Fatalf("Dark: %v", err)
	}
	for bit, lit := range mock.Pattern() {
		if lit {
			t.Errorf("bit %d leuchtet nach Dark", bit)
		}
	}
}

func allRed() [signal.DirectionCount]signal.Aspect {
	return [signal.DirectionCount]signal.Aspect{
		signal.AspectRed, signal.AspectRed, signal.AspectRed, signal.AspectRed,
	}
}

func TestCheckCoversAllDirectionPairs(t *testing.T) {
	for _, a := range signal.Directions() {
		for _, b := range signal.Directions() {
			if a == b {
				continue
			}
			aspects := allRed()
			aspects[a] = signal.AspectGreen
			aspects[b] = signal.AspectGreen

			opposite := (a == signal.North && b == signal.South) || (a == signal.South && b == signal.North) ||
				(a == signal.East && b == signal.West) || (a == signal.West && b == signal.East)
			err := Check(aspects)
			if opposite && err != nil {
				t.Errorf("%s und %s gemeinsam gruen wurde abgewiesen: %v", a, b, err)
			}
			if !opposite && err == nil {
				t.Errorf("%s und %s gemeinsam gruen wurde angenommen", a, b)
			}
			if !opposite && !errors.Is(err, ErrConflict) {
				t.Errorf("Fehler %v, erwartet ErrConflict", err)
			}
		}
	}
}

func TestCheckRejectsOverlappingTransitions(t *testing.T) {
	cases := []struct {
		name    string
		aspects [signal.DirectionCount]signal.Aspect
	}{
		{"gelb gegen kreuzendes gruen", [signal.DirectionCount]signal.Aspect{
			signal.AspectYellow, signal.AspectGreen, signal.AspectRed, signal.AspectRed}},
		{"rotgelb gegen kreuzendes gelb", [signal.DirectionCount]signal.Aspect{
			signal.AspectRed, signal.AspectRedYellow, signal.AspectYellow, signal.AspectRed}},
		{"rotgelb gegen kreuzendes rotgelb", [signal.DirectionCount]signal.Aspect{
			signal.AspectRedYellow, signal.AspectRedYellow, signal.AspectRed, signal.AspectRed}},
		{"dunkler kopf neben freigabe", [signal.DirectionCount]signal.Aspect{
			signal.AspectGreen, signal.AspectOff, signal.AspectRed, signal.AspectRed}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := Check(tc.aspects); !errors.Is(err, ErrConflict) {
				t.Errorf("Zustand wurde mit %v angenommen, erwartet ErrConflict", err)
			}
		})
	}
}

func TestCheckAcceptsRegularStates(t *testing.T) {
	cases := []struct {
		name    string
		aspects [signal.DirectionCount]signal.Aspect
	}{
		{"alles rot", allRed()},
		{"nord und sued gruen", [signal.DirectionCount]signal.Aspect{
			signal.AspectGreen, signal.AspectRed, signal.AspectGreen, signal.AspectRed}},
		{"ost und west gruen", [signal.DirectionCount]signal.Aspect{
			signal.AspectRed, signal.AspectGreen, signal.AspectRed, signal.AspectGreen}},
		{"nord und sued gelb", [signal.DirectionCount]signal.Aspect{
			signal.AspectYellow, signal.AspectRed, signal.AspectYellow, signal.AspectRed}},
		{"ost und west rotgelb", [signal.DirectionCount]signal.Aspect{
			signal.AspectRed, signal.AspectRedYellow, signal.AspectRed, signal.AspectRedYellow}},
		{"notzustand blinkt", [signal.DirectionCount]signal.Aspect{
			signal.AspectYellowFlash, signal.AspectYellowFlash, signal.AspectYellowFlash, signal.AspectYellowFlash}},
		{"alles abgeschaltet", [signal.DirectionCount]signal.Aspect{
			signal.AspectOff, signal.AspectOff, signal.AspectOff, signal.AspectOff}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := Check(tc.aspects); err != nil {
				t.Errorf("zulaessiger Zustand wurde abgewiesen: %v", err)
			}
		})
	}
}

var timing = Timing{
	Yellow:    3 * time.Second,
	AllRed:    2 * time.Second,
	RedYellow: time.Second,
}

var start = time.Date(2026, 4, 1, 7, 0, 0, 0, time.UTC)

func TestMachineRunsCompleteCycle(t *testing.T) {
	m := NewMachine(timing, start)
	now := start

	steps := []struct {
		wait     time.Duration
		endGreen bool
		phase    Phase
		stage    Stage
	}{
		{timing.AllRed, false, PhaseNS, StageRedYellow},
		{timing.RedYellow, false, PhaseNS, StageGreen},
		{5 * time.Second, true, PhaseNS, StageYellow},
		{timing.Yellow, false, PhaseNS, StageAllRed},
		{timing.AllRed, false, PhaseEW, StageRedYellow},
		{timing.RedYellow, false, PhaseEW, StageGreen},
		{5 * time.Second, true, PhaseEW, StageYellow},
		{timing.Yellow, false, PhaseEW, StageAllRed},
		{timing.AllRed, false, PhaseNS, StageRedYellow},
	}

	if got := m.State(); got.Phase != PhaseStartup || got.Stage != StageAllRed {
		t.Fatalf("Startzustand %s, erwartet Start und Allrot", got.Name())
	}

	for i, step := range steps {
		before := now
		now = now.Add(step.wait - time.Millisecond)
		if m.Advance(now, false) {
			t.Fatalf("schritt %d: Wechsel schon nach %s", i, now.Sub(before))
		}
		now = now.Add(time.Millisecond)
		if !m.Advance(now, step.endGreen) {
			t.Fatalf("schritt %d: kein Wechsel nach %s", i, now.Sub(before))
		}
		if got := m.State(); got.Phase != step.phase || got.Stage != step.stage {
			t.Fatalf("schritt %d ergibt %s, erwartet %s_%s", i, got.Name(), step.phase, step.stage)
		}
	}
}

func TestMachineHoldsGreenWithoutStrategy(t *testing.T) {
	m := NewMachine(timing, start)
	now := start.Add(timing.AllRed)
	m.Advance(now, false)
	now = now.Add(timing.RedYellow)
	m.Advance(now, false)
	if m.State().Stage != StageGreen {
		t.Fatalf("Zustand %s, erwartet Gruen", m.State().Name())
	}

	for i := 0; i < 100; i++ {
		now = now.Add(time.Second)
		if m.Advance(now, false) {
			t.Fatalf("Gruen endete nach %s ohne Anforderung", now.Sub(start))
		}
	}
}

func TestStateAspects(t *testing.T) {
	cases := []struct {
		state State
		want  [signal.DirectionCount]signal.Aspect
	}{
		{State{Phase: PhaseNS, Stage: StageGreen},
			[signal.DirectionCount]signal.Aspect{signal.AspectGreen, signal.AspectRed, signal.AspectGreen, signal.AspectRed}},
		{State{Phase: PhaseNS, Stage: StageYellow},
			[signal.DirectionCount]signal.Aspect{signal.AspectYellow, signal.AspectRed, signal.AspectYellow, signal.AspectRed}},
		{State{Phase: PhaseEW, Stage: StageRedYellow},
			[signal.DirectionCount]signal.Aspect{signal.AspectRed, signal.AspectRedYellow, signal.AspectRed, signal.AspectRedYellow}},
		{State{Phase: PhaseEW, Stage: StageAllRed},
			[signal.DirectionCount]signal.Aspect{signal.AspectRed, signal.AspectRed, signal.AspectRed, signal.AspectRed}},
		{State{Phase: PhaseStartup, Stage: StageAllRed},
			[signal.DirectionCount]signal.Aspect{signal.AspectRed, signal.AspectRed, signal.AspectRed, signal.AspectRed}},
	}
	for _, tc := range cases {
		if got := tc.state.Aspects(); got != tc.want {
			t.Errorf("%s ergibt %v, erwartet %v", tc.state.Name(), got, tc.want)
		}
	}
	for _, tc := range cases {
		if err := Check(tc.state.Aspects()); err != nil {
			t.Errorf("%s ist unzulaessig: %v", tc.state.Name(), err)
		}
	}
}

func TestTargetClearedOutsideGreen(t *testing.T) {
	m := NewMachine(timing, start)
	now := start.Add(timing.AllRed)
	m.Advance(now, false)
	now = now.Add(timing.RedYellow)
	m.Advance(now, false)
	m.SetTarget(12 * time.Second)

	now = now.Add(time.Second)
	m.Advance(now, true)
	if got := m.State().Target; got != 0 {
		t.Errorf("Zielgruenzeit %s in %s, erwartet null", got, m.State().Name())
	}
}

func TestFaultEntersFlashState(t *testing.T) {
	m := NewMachine(timing, start)
	m.Fault(start.Add(time.Minute))
	if got := m.State(); got.Phase != PhaseFault {
		t.Fatalf("Zustand %s, erwartet Stoerung", got.Name())
	}
}

func allShow(aspects [signal.DirectionCount]signal.Aspect, want signal.Aspect) bool {
	for _, aspect := range aspects {
		if aspect != want {
			return false
		}
	}
	return true
}

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
	if !allShow(aspectsOf(h.mock.Pattern()), signal.AspectOff) {
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
	if !allShow(first, signal.AspectRed) {
		t.Errorf("das erste Bild nach dem Einschalten ist %v, erwartet Allrot", first)
	}
	h.run(2 * time.Minute)

	previous := aspectsOf(h.mock.History()[mark])
	for i, pattern := range h.mock.History()[mark+1:] {
		current := aspectsOf(pattern)
		if err := Check(current); err != nil {
			t.Fatalf("muster %d ist unzulaessig: %v", i, err)
		}
		for _, direction := range signal.Directions() {
			if !current[direction].CanFollow(previous[direction]) {
				t.Fatalf("muster %d: %s wechselt von %s auf %s",
					i, direction, previous[direction], current[direction])
			}
		}
		previous = current
	}
}

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

func TestFaultSwitchBlinksAndRestarts(t *testing.T) {
	h := newPowerHarness(t)
	h.run(8 * time.Second)

	h.flip(faultPin, true)
	mark := len(h.mock.History())
	h.run(10 * time.Second)

	pulses := 0
	for _, pattern := range h.mock.History()[mark:] {
		switch aspects := aspectsOf(pattern); {
		case allShow(aspects, signal.AspectYellow):
			pulses++
		case allShow(aspects, signal.AspectOff):
		default:
			t.Fatalf("im Notzustand geschriebenes Muster %v", aspects)
		}
	}
	if pulses < 9 || pulses > 11 {
		t.Errorf("%d Gelbimpulse in zehn Sekunden, erwartet etwa zehn", pulses)
	}

	h.flip(faultPin, false)
	first := aspectsOf(h.mock.Pattern())
	if !allShow(first, signal.AspectRed) {
		t.Errorf("nach dem Zuruecklegen zeigt die Kreuzung %v, erwartet Allrot", first)
	}
	h.run(time.Minute)
	if got := h.controller.State(); got.Phase == PhaseFault {
		t.Error("die Anlage blieb im Notzustand")
	}
}

func TestWarningUsesOnlyTheYellowLamp(t *testing.T) {
	h := newPowerHarness(t)
	h.run(8 * time.Second)
	h.flip(faultPin, true)

	mark := len(h.mock.History())
	h.run(3 * time.Second)

	lit := false
	for _, pattern := range h.mock.History()[mark:] {
		for _, direction := range signal.Directions() {
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
