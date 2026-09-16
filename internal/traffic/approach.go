package traffic

import (
	"fmt"

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
	entry     int
}

func NewApproach(direction light.Direction, sensorCount int, queue *detector.Queue) (*Approach, error) {
	if sensorCount <= 0 {
		return nil, fmt.Errorf("zufahrt %s ohne sensoren", direction)
	}
	return &Approach{
		direction: direction,
		occupancy: detector.NewOccupancy(sensorCount),
		queue:     queue,
		entry:     sensorCount - 1,
	}, nil
}

func (a *Approach) Direction() light.Direction { return a.direction }

// QueueLength ist der aktuell geschaetzte Rueckstau in Fahrzeugen.
func (a *Approach) QueueLength() int { return a.queue.Estimate(a.occupancy) }

func (a *Approach) Waiting() int { return a.tracker.Waiting() }

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

// Reset loescht Belegung und wartende Fahrzeuge.
func (a *Approach) Reset() {
	a.occupancy.Reset()
	a.tracker.Reset()
}
