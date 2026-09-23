package controller

import (
	"time"

	"ampel/internal/clock"
	"ampel/internal/detector"
	"ampel/internal/light"
	"ampel/internal/strategy"
)

// DefaultTick ist der Takt der Ereignisschleife.
const DefaultTick = 50 * time.Millisecond

// DefaultFollow ist der groesste Abstand, in dem ein Fahrzeug noch als dicht folgend gilt.
const DefaultFollow = 2 * time.Second

// Setup beschreibt eine Kreuzung in Zahlen. Betrieb und Tests verdrahten damit denselben
// Regelkreis, nur mit anderer Hardware und anderer Uhr.
type Setup struct {
	// Sensors ist je Zufahrt der Kontakt an der Haltelinie.
	Sensors [light.DirectionCount]int
	Follow  time.Duration
	Timing  Timing
	Tick    time.Duration
	Sample  time.Duration

	Strategy strategy.Strategy
	Clock    clock.Clock
	Writer   LampWriter
	Inputs   <-chan Input
	Observer Observer
	Switches *Switches
	Watchdog time.Duration
}

// Build erzeugt Detektor, Ausgabe und Regelkreis.
func Build(setup Setup) (*Controller, error) {
	detect, err := detector.New(setup.Sensors)
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
		Watchdog: setup.Watchdog,
	})
}
