package main

import (
	"fmt"
	"time"

	"ampel/internal/clock"
	"ampel/internal/config"
	"ampel/internal/controller"
	"ampel/internal/hal"
	"ampel/internal/light"
	"ampel/internal/logging"
	"ampel/internal/strategy"
)

// simStep ist der Zeitschritt des Fahrzeugmodells.
const simStep = 50 * time.Millisecond

type simOptions struct {
	config *config.Config
	mode   string
	seed   int64
	rates  [light.DirectionCount]float64
	start  time.Time
	logDir string
}

// greenRecord haelt eine Freigabe mit ihrer Zielgruenzeit fest.
type greenRecord struct {
	phase  controller.Phase
	at     time.Time
	target time.Duration
}

// simulation fuehrt denselben Regelkreis wie der Echtbetrieb, nur mit Mock-Hardware,
// Fake-Uhr und erzeugtem Verkehr.
type simulation struct {
	control  *controller.Controller
	lanes    [light.DirectionCount]*lane
	arrivals *arrivals
	clk      *clock.Fake
	run      *logging.Run
	recorder *logging.Recorder
	mode     string

	greens    []greenRecord
	lastGreen time.Time
}

func newSimulation(options simOptions) (*simulation, error) {
	cfg := options.config
	clk := clock.NewFake(options.start)

	active, err := newStrategy(options.mode, cfg)
	if err != nil {
		return nil, err
	}
	setup, err := cfg.Setup()
	if err != nil {
		return nil, err
	}
	setup.Strategy = active
	setup.Clock = clk
	setup.Writer = hal.NewMock(setup.Bits, 1)
	setup.Tick = simStep

	s := &simulation{
		arrivals: newArrivals(options.seed, options.rates, options.start),
		clk:      clk,
		mode:     active.Name(),
	}
	if options.logDir != "" {
		s.run, err = logging.NewRun(options.logDir, options.start, cfg.Logging.Buffer, active.Name())
		if err != nil {
			return nil, err
		}
		s.recorder = logging.NewRecorder(s.run, options.start, logging.DefaultSettle, active.Name())
		setup.Observer = s.recorder
	}

	s.control, err = controller.Build(setup)
	if err != nil {
		return nil, err
	}
	for _, direction := range light.Directions() {
		s.lanes[direction] = newLane(direction, cfg.Hardware.Sensors.Approaches()[direction], options.start)
	}
	if s.recorder != nil {
		s.recorder.Start(options.start)
	}
	return s, nil
}

// step laesst die simulierte Zeit um einen Schritt verstreichen.
func (s *simulation) step() {
	s.clk.Advance(simStep)
	now := s.clk.Now()

	for _, direction := range s.arrivals.due(now) {
		s.lanes[direction].arrive(now)
	}
	state := s.control.State()
	for _, l := range s.lanes {
		released := state.Stage == controller.StageGreen && releases(state.Phase, l.direction)
		for _, input := range l.step(now, released) {
			s.control.Feed(input)
		}
	}
	s.control.Step(now)
	s.recordGreen()
}

// recordGreen merkt sich jede neue Freigabe mit ihrer Zielgruenzeit.
func (s *simulation) recordGreen() {
	state := s.control.State()
	if state.Stage != controller.StageGreen || state.Since.Equal(s.lastGreen) {
		return
	}
	s.lastGreen = state.Since
	s.greens = append(s.greens, greenRecord{phase: state.Phase, at: state.Since, target: state.Target})
}

// meanTarget ist die mittlere Zielgruenzeit einer Phase innerhalb der ersten window.
func (s *simulation) meanTarget(phase controller.Phase, window time.Duration) time.Duration {
	var sum time.Duration
	var count int
	for _, green := range s.greens {
		if green.phase != phase || green.at.Sub(s.greens[0].at) > window {
			continue
		}
		sum += green.target
		count++
	}
	if count == 0 {
		return 0
	}
	return sum / time.Duration(count)
}

// run laeuft die angegebene simulierte Zeit. show wird, wenn gesetzt, jede simulierte
// Sekunde aufgerufen.
func (s *simulation) walk(duration time.Duration, show func(*simulation)) {
	next := s.clk.Now().Add(time.Second)
	for elapsed := time.Duration(0); elapsed < duration; elapsed += simStep {
		s.step()
		if show != nil && !s.clk.Now().Before(next) {
			next = s.clk.Now().Add(time.Second)
			show(s)
		}
	}
}

func (s *simulation) close() error {
	if s.recorder != nil {
		s.recorder.Stop(s.clk.Now())
	}
	if err := s.control.Shutdown(); err != nil {
		return err
	}
	if s.run != nil {
		return s.run.Close()
	}
	return nil
}

// result sind die Kennzahlen eines Laufs.
type result struct {
	mode     string
	arrived  int
	departed int
	measured time.Duration
	truth    time.Duration
	worst    time.Duration
}

func (s *simulation) result() result {
	r := result{mode: s.mode, measured: s.control.Metrics().MeanAll()}
	var sum time.Duration
	for _, l := range s.lanes {
		r.arrived += l.arrived
		r.departed += l.departed
		sum += l.waitSum
		if l.waitMax > r.worst {
			r.worst = l.waitMax
		}
	}
	if r.departed > 0 {
		r.truth = sum / time.Duration(r.departed)
	}
	return r
}

func (r result) String() string {
	return fmt.Sprintf("%-9s Ankuenfte %5d  Abfahrten %5d  Wartezeit gemessen %7s  Modell %7s  Maximum %7s",
		r.mode, r.arrived, r.departed, round(r.measured), round(r.truth), round(r.worst))
}

func round(d time.Duration) time.Duration { return d.Round(10 * time.Millisecond) }

func newStrategy(mode string, cfg *config.Config) (strategy.Strategy, error) {
	switch mode {
	case "festzeit":
		return strategy.NewFixed(cfg.Fixed.Green.Duration()), nil
	case "adaptiv":
		return cfg.Following()
	}
	return nil, fmt.Errorf("unbekannte Betriebsart %q, erlaubt sind festzeit und adaptiv", mode)
}

func releases(phase controller.Phase, direction light.Direction) bool {
	for _, released := range phase.Directions() {
		if released == direction {
			return true
		}
	}
	return false
}
