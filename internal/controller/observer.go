package controller

import (
	"ampel/internal/detector"
	"ampel/internal/light"
	"time"
)

type Snapshot struct {
	State     State
	Elapsed   time.Duration
	Following int
	Aspects   [light.DirectionCount]light.Aspect
	Green     [light.DirectionCount]time.Duration
}

type Observer interface {
	PhaseChanged(at time.Time, state State)
	SensorChanged(event detector.SensorEvent)
	PowerChanged(at time.Time, on bool)
	Fault(at time.Time, err error)
	Sample(at time.Time, snapshot Snapshot)
}

type NopObserver struct{}

var _ Observer = NopObserver{}

func (NopObserver) PhaseChanged(time.Time, State)      {}
func (NopObserver) SensorChanged(detector.SensorEvent) {}
func (NopObserver) PowerChanged(time.Time, bool)       {}
func (NopObserver) Fault(time.Time, error)             {}
func (NopObserver) Sample(time.Time, Snapshot)         {}
