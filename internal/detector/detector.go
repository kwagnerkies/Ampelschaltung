package detector

import (
	"fmt"
	"time"

	"ampel/internal/light"
)

// Detector ordnet GPIO-Pins den Zufahrten zu. Entprellt wird im Kernel, nicht hier: der
// Treiber kann das zuverlaessiger, und eine zweite Stufe in Go wuerde dieselbe Arbeit
// doppelt machen.
type Detector struct {
	sensors map[int]*sensor
}

type sensor struct {
	direction light.Direction
	index     int
	occupied  bool
}

// New erwartet die Sensorpins je Zufahrt in der Reihenfolge Haltelinie, dann aufwaerts.
func New(pins [light.DirectionCount][]int) (*Detector, error) {
	d := &Detector{sensors: make(map[int]*sensor)}
	for direction, approach := range pins {
		for index, pin := range approach {
			if _, taken := d.sensors[pin]; taken {
				return nil, fmt.Errorf("BCM %d ist doppelt zugeordnet", pin)
			}
			d.sensors[pin] = &sensor{direction: light.Direction(direction), index: index}
		}
	}
	if len(d.sensors) == 0 {
		return nil, fmt.Errorf("kein sensor zugeordnet")
	}
	return d, nil
}

// Feed nimmt eine Flanke auf. Ein wiederholter Pegel erzeugt kein Ereignis.
func (d *Detector) Feed(pin int, occupied bool, at time.Time) ([]SensorEvent, error) {
	s, ok := d.sensors[pin]
	if !ok {
		return nil, fmt.Errorf("BCM %d gehoert zu keiner zufahrt", pin)
	}
	if occupied == s.occupied {
		return nil, nil
	}
	s.occupied = occupied
	return []SensorEvent{{
		Direction: s.direction,
		Index:     s.index,
		Occupied:  occupied,
		At:        at,
	}}, nil
}

// Reset vergisst alle Pegel. Danach gilt jede Zufahrt als frei.
func (d *Detector) Reset() {
	for _, s := range d.sensors {
		s.occupied = false
	}
}

// Knows sagt, ob dieser Pin zu einem Sensor gehoert. Der Hauptschalter gehoert nicht dazu.
func (d *Detector) Knows(pin int) bool {
	_, ok := d.sensors[pin]
	return ok
}
