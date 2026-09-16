package controller

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ampel/internal/clock"
	"ampel/internal/detector"
	"ampel/internal/light"
	"ampel/internal/strategy"
	"ampel/internal/traffic"
)

// Input ist eine Flanke an einem Eingang. Der Typ steht hier, damit der Regelkreis die
// Hardwareschicht nicht kennt.
type Input struct {
	Pin    int
	Active bool
	Time   time.Time
}

// Options buendelt, was der Regelkreis zum Laufen braucht.
type Options struct {
	Timing     Timing
	Tick       time.Duration
	Sample     time.Duration
	Detector   *detector.Detector
	Approaches [light.DirectionCount]*traffic.Approach
	Output     *Output
	Strategy   strategy.Strategy
	Clock      clock.Clock
	Inputs     <-chan Input
	Observer   Observer
	// FlashHalf ist die halbe Periode des Gelbblinkens im Notzustand.
	FlashHalf time.Duration
	// Power ist der Hauptschalter. Ohne ihn laeuft die Anlage immer, wie es der Simulator
	// braucht.
	Power *Power
	// Watchdog ist die groesste erlaubte Pause zwischen zwei Takten.
	Watchdog time.Duration
	// Follow ist der groesste Abstand, in dem ein Fahrzeug noch als dicht folgend gilt.
	Follow time.Duration
}

// Controller ist der Regelkreis. Ein einziger Goroutine besitzt diesen Zustand; alles andere
// kommuniziert ueber Kanaele.
type Controller struct {
	timing     Timing
	tick       time.Duration
	sample     time.Duration
	flashHalf  time.Duration
	machine    *Machine
	detect     *detector.Detector
	approaches [light.DirectionCount]*traffic.Approach
	output     *Output
	strategy   strategy.Strategy
	clk        clock.Clock
	inputs     <-chan Input
	observer   Observer
	metrics    traffic.Metrics
	power      *power
	watchdog   *Watchdog

	follow time.Duration

	started time.Time
	// lastCrossing ist je Zufahrt die letzte Ueberfahrt der Haltelinie in dieser Freigabe,
	// following die Zahl der dicht darauf folgenden Fahrzeuge.
	lastCrossing [light.DirectionCount]time.Time
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
	for direction, approach := range options.Approaches {
		if approach == nil {
			return nil, fmt.Errorf("regelkreis: zufahrt %s fehlt", light.Direction(direction))
		}
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
		approaches: options.Approaches,
		output:     options.Output,
		strategy:   options.Strategy,
		follow:     options.Follow,
		clk:        options.Clock,
		inputs:     options.Inputs,
		observer:   options.Observer,
		started:    now,
		lastSample: now,
		watchdog:   NewWatchdog(options.Watchdog, now),
	}
	if options.Power != nil {
		c.power = newPower(*options.Power, now)
	}
	return c, nil
}

func (c *Controller) State() State { return c.machine.State() }

func (c *Controller) Metrics() *traffic.Metrics { return &c.metrics }

func (c *Controller) Mode() string { return c.strategy.Name() }

// Begin zeigt das Startbild. Der Aufruf ist mehrfach moeglich und wirkt nur einmal; Step
// holt ihn nach, damit kein Aufrufer ohne Signalbild losfaehrt.
func (c *Controller) Begin(now time.Time) error {
	if c.begun {
		return nil
	}
	c.begun = true
	if err := c.show(); err != nil {
		return err
	}
	c.observer.PhaseChanged(now, c.machine.State(), c.strategy.Name())
	return nil
}

// Run haelt die Kreuzung in Betrieb, bis der Kontext endet. Danach steht alles auf Rot.
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

// Step fuehrt einen Schritt des Regelkreises aus. Der Simulator ruft das direkt auf, damit
// er ohne Ticker und ohne echte Zeit laufen kann.
func (c *Controller) Step(now time.Time) {
	if err := c.Begin(now); err != nil {
		c.enterFault(now, err)
		return
	}
	// Der Hauptschalter wird vor dem Notzustand abgefragt: aus und wieder an ist der
	// Neustart, den die Sicherheitsregel nach einer Stoerung verlangt.
	if c.powerStep(now) {
		return
	}
	if c.machine.State().Phase == PhaseFault {
		c.flash(now)
		return
	}
	if gap, late := c.watchdog.Kick(now); late {
		c.enterFault(now, fmt.Errorf("regelkreis hat %s nicht getaktet, erlaubt sind %s",
			gap.Round(time.Millisecond), c.watchdog.Limit()))
		return
	}
	c.applyEvents(c.detect.Tick(now))

	state := c.machine.State()
	endGreen := state.Stage == StageGreen && c.strategy.EndGreen(c.view(now))
	if c.machine.Advance(now, endGreen) {
		state = c.machine.State()
		if state.Stage == StageGreen {
			c.following = 0
			c.lastCrossing = [light.DirectionCount]time.Time{}
			state = c.machine.State()
		}
		if err := c.show(); err != nil {
			c.enterFault(now, err)
			return
		}
		c.observer.PhaseChanged(now, state, c.strategy.Name())
	}
	// Die Zielzeit waechst mit jedem dicht folgenden Fahrzeug, deshalb wird sie in jedem Takt
	// nachgefuehrt. Anzeige und Log zeigen damit immer den geltenden Wert.
	if c.machine.State().Stage == StageGreen {
		c.machine.SetTarget(c.strategy.TargetGreen(c.view(now)))
	}
	c.emitSample(now)
}

// Reset verwirft Belegung, wartende Fahrzeuge und Kennzahlen und beginnt damit eine neue
// Messung. Der Reset-Taster loest das aus.
func (c *Controller) Reset(now time.Time) {
	c.detect.Reset()
	for _, approach := range c.approaches {
		approach.Reset()
	}
	c.metrics.Reset()
	c.following = 0
	c.lastCrossing = [light.DirectionCount]time.Time{}
}
