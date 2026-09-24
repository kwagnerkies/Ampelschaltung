package steuerung

import (
	"ampel/src/erkennung"
	"ampel/src/signal"
	"time"
)

type Snapshot struct {
	State     State
	Elapsed   time.Duration
	Following int
	Aspects   [signal.DirectionCount]signal.Aspect
	Green     [signal.DirectionCount]time.Duration
}

type Observer interface {
	PhaseChanged(at time.Time, state State)
	SensorChanged(event erkennung.SensorEvent)
	PowerChanged(at time.Time, on bool)
	Fault(at time.Time, err error)
	Sample(at time.Time, snapshot Snapshot)
}

type NopObserver struct{}

var _ Observer = NopObserver{}

func (NopObserver) PhaseChanged(time.Time, State)       {}
func (NopObserver) SensorChanged(erkennung.SensorEvent) {}
func (NopObserver) PowerChanged(time.Time, bool)        {}
func (NopObserver) Fault(time.Time, error)              {}
func (NopObserver) Sample(time.Time, Snapshot)          {}

type Observers []Observer

var _ Observer = Observers{}

func (o Observers) PhaseChanged(at time.Time, state State) {
	for _, observer := range o {
		observer.PhaseChanged(at, state)
	}
}

func (o Observers) SensorChanged(event erkennung.SensorEvent) {
	for _, observer := range o {
		observer.SensorChanged(event)
	}
}

func (o Observers) PowerChanged(at time.Time, on bool) {
	for _, observer := range o {
		observer.PowerChanged(at, on)
	}
}

func (o Observers) Fault(at time.Time, err error) {
	for _, observer := range o {
		observer.Fault(at, err)
	}
}

func (o Observers) Sample(at time.Time, snapshot Snapshot) {
	for _, observer := range o {
		observer.Sample(at, snapshot)
	}
}
