package controller

import (
	"time"

	"ampel/internal/detector"
	"ampel/internal/light"
)

// Snapshot ist der Abtastwert des Regelkreises fuer Zustandslog und Anzeige.
type Snapshot struct {
	State   State
	Elapsed time.Duration
	Mode    string
	// Reach ist je Zufahrt die Zahl belegter Sensoren.
	Reach [light.DirectionCount]int
	// Following ist die Zahl der Verlaengerungen in der laufenden Freigabe.
	Following int
	// Aspects ist das aktuelle Signalbild je Zufahrt.
	Aspects [light.DirectionCount]light.Aspect
	// Green ist je Zufahrt die Gruenzeit, die zaehlt: bei laufender Freigabe die
	// beschlossene, sonst die Vorschau aus der aktuellen Nachfrage. Ohne die Vorschau wuerde
	// eine Anzeige erst beim Phasenwechsel auf ein Fahrzeug reagieren.
	Green [light.DirectionCount]time.Duration
}

// Observer sieht dem Regelkreis zu. Hier haengt sich das CSV-Logging ein.
type Observer interface {
	PhaseChanged(at time.Time, state State, mode string)
	SensorChanged(event detector.SensorEvent)
	// PowerChanged meldet den Hauptschalter.
	PowerChanged(at time.Time, on bool)
	Fault(at time.Time, err error)
	Sample(at time.Time, snapshot Snapshot)
}

// NopObserver verwirft alles. Damit laeuft der Regelkreis auch ohne Logging.
type NopObserver struct{}

var _ Observer = NopObserver{}

func (NopObserver) PhaseChanged(time.Time, State, string) {}
func (NopObserver) SensorChanged(detector.SensorEvent)    {}
func (NopObserver) PowerChanged(time.Time, bool)          {}
func (NopObserver) Fault(time.Time, error)                {}
func (NopObserver) Sample(time.Time, Snapshot)            {}
