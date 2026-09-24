package steuerung

import (
	"ampel/src/clock"
	"ampel/src/erkennung"
	"ampel/src/regel"
	"ampel/src/signal"
	"context"
	"errors"
	"fmt"
	"time"
)

type Input struct {
	Pin    int
	Active bool
	Time   time.Time
}

type Options struct {
	Timing    Timing
	Tick      time.Duration
	Sample    time.Duration
	Detector  *erkennung.Detector
	Output    *Output
	Strategy  regel.Strategy
	Clock     clock.Clock
	Inputs    <-chan Input
	Observer  Observer
	FlashHalf time.Duration
	Switches  *Switches
	Follow    time.Duration
}

type Controller struct {
	timing    Timing
	tick      time.Duration
	sample    time.Duration
	flashHalf time.Duration
	machine   *Machine
	detect    *erkennung.Detector
	output    *Output
	strategy  regel.Strategy
	clk       clock.Clock
	inputs    <-chan Input
	observer  Observer
	switches  *switches

	follow time.Duration

	started      time.Time
	lastCrossing [signal.DirectionCount]time.Time
	following    int
	lastSample   time.Time
	begun        bool
	flashOn      bool
	fault        error
}

func New(options Options) (*Controller, error) {
	if options.Detector == nil || options.Output == nil || options.Strategy == nil || options.Clock == nil {
		return nil, errors.New("regelkreis: detektor, ausgabe, strategie und uhr sind pflicht")
	}
	if options.Tick <= 0 {
		return nil, fmt.Errorf("regelkreis: takt %s", options.Tick)
	}
	if options.Observer == nil {
		options.Observer = NopObserver{}
	}
	if options.Follow <= 0 {
		options.Follow = DefaultFollow
	}
	if options.FlashHalf <= 0 {
		options.FlashHalf = 500 * time.Millisecond
	}
	now := options.Clock.Now()
	c := &Controller{
		timing:     options.Timing,
		tick:       options.Tick,
		sample:     options.Sample,
		flashHalf:  options.FlashHalf,
		machine:    NewMachine(options.Timing, now),
		detect:     options.Detector,
		output:     options.Output,
		strategy:   options.Strategy,
		follow:     options.Follow,
		clk:        options.Clock,
		inputs:     options.Inputs,
		observer:   options.Observer,
		started:    now,
		lastSample: now,
	}
	if options.Switches != nil {
		c.switches = newSwitches(*options.Switches, now)
	}
	return c, nil
}

func (c *Controller) State() State { return c.machine.State() }

func (c *Controller) Begin(now time.Time) error {
	if c.begun {
		return nil
	}
	c.begun = true
	if err := c.show(); err != nil {
		return err
	}
	c.observer.PhaseChanged(now, c.machine.State())
	return nil
}

func (c *Controller) Run(ctx context.Context) error {
	now := c.clk.Now()
	if err := c.Begin(now); err != nil {
		return err
	}

	ticker := c.clk.Ticker(c.tick)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return errors.Join(c.Shutdown(), c.fault)
		case input := <-c.inputs:
			c.Feed(input)
		case <-ticker.C():
			c.Step(c.clk.Now())
		}
	}
}

func (c *Controller) Step(now time.Time) {
	if err := c.Begin(now); err != nil {
		c.enterFault(now, err)
		return
	}
	if c.switchStep(now) {
		return
	}
	if c.machine.State().Phase == PhaseFault {
		c.flash(now)
		return
	}

	state := c.machine.State()
	endGreen := state.Stage == StageGreen && c.strategy.EndGreen(c.view(now))
	if c.machine.Advance(now, endGreen) {
		state = c.machine.State()
		if state.Stage == StageGreen {
			c.following = 0
			c.lastCrossing = [signal.DirectionCount]time.Time{}
			state = c.machine.State()
		}
		if err := c.show(); err != nil {
			c.enterFault(now, err)
			return
		}
		c.observer.PhaseChanged(now, state)
	}
	if c.machine.State().Stage == StageGreen {
		c.machine.SetTarget(c.strategy.TargetGreen(c.view(now)))
	}
	c.emitSample(now)
}

func (c *Controller) Reset(now time.Time) {
	c.detect.Reset()
	c.following = 0
	c.lastCrossing = [signal.DirectionCount]time.Time{}
}

const DefaultTick = 50 * time.Millisecond

const DefaultFollow = 2 * time.Second

type Setup struct {
	Sensors [signal.DirectionCount]int
	Follow  time.Duration
	Timing  Timing
	Tick    time.Duration
	Sample  time.Duration

	Strategy regel.Strategy
	Clock    clock.Clock
	Writer   LampWriter
	Inputs   <-chan Input
	Observer Observer
	Switches *Switches
}

func Build(setup Setup) (*Controller, error) {
	detect, err := erkennung.New(setup.Sensors)
	if err != nil {
		return nil, err
	}
	if setup.Tick <= 0 {
		setup.Tick = DefaultTick
	}
	return New(Options{
		Timing:   setup.Timing,
		Tick:     setup.Tick,
		Sample:   setup.Sample,
		Detector: detect,
		Output:   NewOutput(setup.Writer),
		Strategy: setup.Strategy,
		Follow:   setup.Follow,
		Clock:    setup.Clock,
		Inputs:   setup.Inputs,
		Observer: setup.Observer,
		Switches: setup.Switches,
	})
}

func (c *Controller) Feed(input Input) {
	if c.switches != nil && c.switches.knows(input.Pin) {
		c.switches.level(input.Pin, input.Active)
		return
	}
	if !c.On() || !c.detect.Knows(input.Pin) {
		return
	}
	events, err := c.detect.Feed(input.Pin, input.Active, input.Time)
	if err != nil {
		return
	}
	c.applyEvents(events)
}

func (c *Controller) applyEvents(events []erkennung.SensorEvent) {
	for _, event := range events {
		c.observer.SensorChanged(event)
		if !event.Occupied {
			c.countCrossing(event.At, event.Direction)
		}
	}
}

func (c *Controller) view(now time.Time) regel.View {
	return regel.View{
		Now:        now,
		GreenSince: c.machine.State().Since,
		Following:  c.following,
	}
}

func (c *Controller) countCrossing(at time.Time, direction signal.Direction) {
	state := c.machine.State()
	if state.Stage != StageGreen || PhaseOf(direction) != state.Phase {
		return
	}
	if last := c.lastCrossing[direction]; !last.IsZero() && at.Sub(last) <= c.follow {
		c.following++
	}
	c.lastCrossing[direction] = at
}

func (c *Controller) Snapshot(now time.Time) Snapshot {
	state := c.machine.State()
	snapshot := Snapshot{
		State:     state,
		Elapsed:   now.Sub(state.Since),
		Following: c.following,
		Aspects:   state.Aspects(),
	}
	for _, direction := range signal.Directions() {
		if PhaseOf(direction) == state.Phase && state.Stage == StageGreen {
			if remaining := state.Target - snapshot.Elapsed; remaining > 0 {
				snapshot.Green[direction] = remaining
			}
		} else {
			snapshot.Green[direction] = c.strategy.TargetGreen(regel.View{Now: now, GreenSince: now})
		}
	}
	return snapshot
}
