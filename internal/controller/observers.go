package controller

import (
	"time"

	"ampel/internal/detector"
)

// Observers verteilt jedes Ereignis an mehrere Beobachter, damit Logging und Anzeige
// nebeneinander am Regelkreis haengen koennen. Aufgerufen wird in der Reihenfolge der Liste,
// aus der Goroutine des Regelkreises.
type Observers []Observer

var _ Observer = Observers{}

func (o Observers) PhaseChanged(at time.Time, state State, mode string) {
	for _, observer := range o {
		observer.PhaseChanged(at, state, mode)
	}
}

func (o Observers) SensorChanged(event detector.SensorEvent) {
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
