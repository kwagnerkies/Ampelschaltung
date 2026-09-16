package main

import (
	"fmt"
	"time"

	"ampel/internal/clock"
	"ampel/internal/config"
	"ampel/internal/controller"
	"ampel/internal/hal"
	"ampel/internal/light"
)

// simStep ist der Zeitschritt des Fahrzeugmodells.
const simStep = 50 * time.Millisecond

type simOptions struct {
	config *config.Config
	seed   int64
	rates  [light.DirectionCount]float64
	start  time.Time
}

// simulation fuehrt denselben Regelkreis wie der Echtbetrieb, nur mit Mock-Hardware,
// Fake-Uhr und erzeugtem Verkehr.
type simulation struct {
	control  *controller.Controller
	lanes    [light.DirectionCount]*lane
	arrivals *arrivals
	clk      *clock.Fake
}

func newSimulation(options simOptions) (*simulation, error) {
	cfg := options.config
	clk := clock.NewFake(options.start)

	setup, err := cfg.Setup()
	if err != nil {
		return nil, err
	}
	setup.Strategy, err = cfg.Following()
	if err != nil {
		return nil, err
	}
	setup.Clock = clk
	setup.Writer = hal.NewMock(setup.Bits, 1)
	setup.Tick = simStep

	s := &simulation{
		arrivals: newArrivals(options.seed, options.rates, options.start),
		clk:      clk,
	}
	s.control, err = controller.Build(setup)
	if err != nil {
		return nil, err
	}
	for _, direction := range light.Directions() {
		s.lanes[direction] = newLane(direction, cfg.Hardware.Sensors.Approaches()[direction], options.start)
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
}

// walk laeuft die angegebene simulierte Zeit. show wird, wenn gesetzt, jede simulierte
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

func (s *simulation) close() error { return s.control.Shutdown() }

// result ist der Ueberblick eines Laufs. Die Zahlen stammen aus dem Fahrzeugmodell des
// Simulators, nicht aus der Steuerung.
type result struct {
	arrived  int
	departed int
	mean     time.Duration
	worst    time.Duration
}

func (s *simulation) result() result {
	var r result
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
		r.mean = sum / time.Duration(r.departed)
	}
	return r
}

func (r result) String() string {
	return fmt.Sprintf("Ankuenfte %5d  Abfahrten %5d  Wartezeit im Modell %7s  Maximum %7s",
		r.arrived, r.departed, round(r.mean), round(r.worst))
}

func round(d time.Duration) time.Duration { return d.Round(10 * time.Millisecond) }

func releases(phase controller.Phase, direction light.Direction) bool {
	for _, released := range phase.Directions() {
		if released == direction {
			return true
		}
	}
	return false
}
