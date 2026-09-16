package controller

import (
	"time"

	"ampel/internal/detector"
	"ampel/internal/light"
	"ampel/internal/traffic"
)

// Snapshot ist der Abtastwert des Regelkreises fuer das Zustandslog.
type Snapshot struct {
	State   State
	Elapsed time.Duration
	Mode    string
	Queues  [light.DirectionCount]int
	Demands [light.DirectionCount]float64
	Weight  float64
}

// Learner mischt Messung und Prognose und lernt das Tagesprofil mit.
type Learner interface {
	// Blend mischt die gemessene Nachfrage mit der Prognose und liefert deren Gewicht.
	Blend(at time.Time, direction light.Direction, live float64) (float64, float64)
	// Observe uebernimmt die gemessene Nachfrage am Ende einer Freigabe.
	Observe(at time.Time, direction light.Direction, demand float64)
	// Persist sichert den Lernzustand, wenn es Zeit dafuer ist.
	Persist(at time.Time)
	Reset()
}

// NopLearner lernt nichts und gibt die Messung unveraendert weiter. Das ist das Verhalten im
// Festzeitbetrieb.
type NopLearner struct{}

var _ Learner = NopLearner{}

func (NopLearner) Blend(_ time.Time, _ light.Direction, live float64) (float64, float64) {
	return live, 0
}

func (NopLearner) Observe(time.Time, light.Direction, float64) {}

func (NopLearner) Persist(time.Time) {}

func (NopLearner) Reset() {}

// Observer sieht dem Regelkreis zu. Hier haengt sich das CSV-Logging ein.
type Observer interface {
	PhaseChanged(at time.Time, state State, mode string)
	SensorChanged(event detector.SensorEvent)
	VehicleLeft(departure traffic.Departure, mode string, phase string)
	ModeChanged(at time.Time, mode string)
	// Reset meldet den geloeschten Lernzustand. Das Logging beginnt daraufhin einen neuen Lauf.
	Reset(at time.Time)
	Fault(at time.Time, err error)
	Sample(at time.Time, snapshot Snapshot)
}

// NopObserver verwirft alles. Damit laeuft der Regelkreis auch ohne Logging.
type NopObserver struct{}

var _ Observer = NopObserver{}

func (NopObserver) PhaseChanged(time.Time, State, string)         {}
func (NopObserver) SensorChanged(detector.SensorEvent)            {}
func (NopObserver) VehicleLeft(traffic.Departure, string, string) {}
func (NopObserver) ModeChanged(time.Time, string)                 {}
func (NopObserver) Reset(time.Time)                               {}
func (NopObserver) Fault(time.Time, error)                        {}
func (NopObserver) Sample(time.Time, Snapshot)                    {}
