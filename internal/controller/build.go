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

// Setup beschreibt eine Kreuzung in Zahlen. Betrieb und Simulator verdrahten damit denselben
// Regelkreis, nur mit anderer Hardware und anderer Uhr.
type Setup struct {
	Sensors      [light.DirectionCount][]int
	Debounce     time.Duration
	QueueMapping map[int]int
	DemandAlpha  float64
	LampMatrix   [light.DirectionCount][3]int
	Bits         int
	Timing       Timing
	Tick         time.Duration
	Sample       time.Duration

	Strategy strategy.Strategy
	Learner  Learner
	Clock    clock.Clock
	Writer   LampWriter
	Inputs   <-chan Input
	Observer Observer
	Panel    *Panel
	Watchdog time.Duration
}

// Build erzeugt Detektor, Zufahrten, Lampenbus und Regelkreis.
func Build(setup Setup) (*Controller, error) {
	detect, err := detector.New(setup.Sensors, setup.Debounce)
	if err != nil {
		return nil, err
	}
	sensorCount := len(setup.Sensors[light.North])
	queue, err := detector.NewQueue(setup.QueueMapping, sensorCount)
	if err != nil {
		return nil, err
	}
	var approaches [light.DirectionCount]*traffic.Approach
	for _, direction := range light.Directions() {
		if got := len(setup.Sensors[direction]); got != sensorCount {
			return nil, fmt.Errorf("zufahrt %s hat %d sensoren, nord hat %d", direction, got, sensorCount)
		}
		approaches[direction], err = traffic.NewApproach(direction, sensorCount, queue, setup.DemandAlpha)
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
		Learner:    setup.Learner,
		Clock:      setup.Clock,
		Inputs:     setup.Inputs,
		Observer:   setup.Observer,
		Panel:      setup.Panel,
		Watchdog:   setup.Watchdog,
	})
}
