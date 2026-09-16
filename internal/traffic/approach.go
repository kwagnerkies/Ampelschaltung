package traffic

import (
	"fmt"
	"time"

	"ampel/internal/detector"
	"ampel/internal/light"
)

// Approach ist der Zustand einer Zufahrt: Belegung, geschaetzter Rueckstau, geglaettete
// Nachfrage und die wartenden Fahrzeuge.
type Approach struct {
	direction light.Direction
	occupancy *detector.Occupancy
	queue     *detector.Queue
	tracker   Tracker
	alpha     float64
	demand    float64
	entry     int
}

// NewApproach erwartet die geglaettete Gewichtung alpha aus der Konfiguration.
func NewApproach(direction light.Direction, sensorCount int, queue *detector.Queue, alpha float64) (*Approach, error) {
	if sensorCount <= 0 {
		return nil, fmt.Errorf("zufahrt %s ohne sensoren", direction)
	}
	if alpha <= 0 || alpha > 1 {
		return nil, fmt.Errorf("zufahrt %s: alpha %v liegt nicht zwischen null und eins", direction, alpha)
	}
	return &Approach{
		direction: direction,
		occupancy: detector.NewOccupancy(sensorCount),
		queue:     queue,
		alpha:     alpha,
		entry:     sensorCount - 1,
	}, nil
}

func (a *Approach) Direction() light.Direction { return a.direction }

// QueueLength ist der aktuell geschaetzte Rueckstau in Fahrzeugen.
func (a *Approach) QueueLength() int { return a.queue.Estimate(a.occupancy) }

// Demand ist das geglaettete Mittel der Rueckstaulaenge.
func (a *Approach) Demand() float64 { return a.demand }

func (a *Approach) Waiting() int { return a.tracker.Waiting() }

func (a *Approach) OldestWait(now time.Time) time.Duration { return a.tracker.OldestWait(now) }

func (a *Approach) AtStopLine() bool { return a.occupancy.AtStopLine() }

// Apply verarbeitet ein Sensorereignis. Ein Fahrzeug gilt als angekommen, wenn es den
// hintersten Sensor der Erfassung erreicht, und als abgefahren, wenn es die Haltelinie
// wieder freigibt. Erreicht ein Fahrzeug die Haltelinie, ohne vorher erfasst worden zu sein,
// zaehlt der Beginn an der Linie als Ankunft.
func (a *Approach) Apply(event detector.SensorEvent, phase int) (Departure, bool) {
	a.occupancy.Apply(event.Index, event.Occupied, event.At)

	switch {
	case event.Occupied && event.Index == a.entry:
		a.tracker.Arrive(Arrival{At: event.At, Queue: a.QueueLength(), Phase: phase})
	case event.Occupied && event.Index == 0 && a.tracker.Waiting() == 0:
		a.tracker.Arrive(Arrival{At: event.At, Queue: a.QueueLength(), Phase: phase})
	case !event.Occupied && event.Index == 0:
		return a.tracker.Depart(a.direction, event.At)
	}
	return Departure{}, false
}

// UpdateDemand glaettet die Nachfrage. Der Regelkreis ruft das am Ende jeder Phase auf.
func (a *Approach) UpdateDemand() {
	a.demand = a.alpha*float64(a.QueueLength()) + (1-a.alpha)*a.demand
}

// Reset loescht Belegung, wartende Fahrzeuge und die geglaettete Nachfrage.
func (a *Approach) Reset() {
	a.occupancy.Reset()
	a.tracker.Reset()
	a.demand = 0
}
