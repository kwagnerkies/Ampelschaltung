package controller

import (
	"fmt"
	"time"

	"ampel/internal/clock"
	"ampel/internal/detector"
	"ampel/internal/light"
	"ampel/internal/strategy"
	"ampel/internal/traffic"
)

// DefaultTick ist der Takt der Ereignisschleife.
const DefaultTick = 50 * time.Millisecond

// DefaultFollow ist der groesste Abstand, in dem ein Fahrzeug noch als dicht folgend gilt.
const DefaultFollow = 2 * time.Second

// Setup beschreibt eine Kreuzung in Zahlen. Betrieb und Simulator verdrahten damit denselben
// Regelkreis, nur mit anderer Hardware und anderer Uhr.
type Setup struct {
	Sensors    [light.DirectionCount][]int
	Debounce   time.Duration
	Follow     time.Duration
	LampMatrix [light.DirectionCount][3]int
	Bits       int
	Timing     Timing
	Tick       time.Duration
	Sample     time.Duration

	Strategy strategy.Strategy
	Clock    clock.Clock
	Writer   LampWriter
	Inputs   <-chan Input
	Observer Observer
	Power    *Power
	Watchdog time.Duration
}

// Build erzeugt Detektor, Zufahrten, Lampenbus und Regelkreis.
func Build(setup Setup) (*Controller, error) {
	detect, err := detector.New(setup.Sensors)
	if err != nil {
		return nil, err
	}
	sensorCount := len(setup.Sensors[light.North])
	var approaches [light.DirectionCount]*traffic.Approach
	for _, direction := range light.Directions() {
		if got := len(setup.Sensors[direction]); got != sensorCount {
			return nil, fmt.Errorf("zufahrt %s hat %d sensoren, nord hat %d", direction, got, sensorCount)
		}
		approaches[direction], err = traffic.NewApproach(direction, sensorCount)
		if err != nil {
			return nil, err
		}
	}
	bus, err := light.NewBus(setup.LampMatrix, setup.Bits)
	if err != nil {
		return nil, err
	}
	if setup.Tick <= 0 {
		setup.Tick = DefaultTick
	}
	return New(Options{
		Timing:     setup.Timing,
		Tick:       setup.Tick,
		Sample:     setup.Sample,
		Detector:   detect,
		Approaches: approaches,
		Output:     NewOutput(bus, setup.Writer),
		Strategy:   setup.Strategy,
		Follow:     setup.Follow,
		Clock:      setup.Clock,
		Inputs:     setup.Inputs,
		Observer:   setup.Observer,
		Power:      setup.Power,
		Watchdog:   setup.Watchdog,
	})
}
