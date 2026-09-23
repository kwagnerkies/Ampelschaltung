// Paket detector ordnet die Haltelinien-Kontakte den Zufahrten zu.
package detector

import (
	"fmt"
	"time"

	"ampel/internal/light"
)

// Detector ordnet GPIO-Pins den Zufahrten zu. Entprellt wird im Kernel, nicht hier: der
// Treiber kann das zuverlaessiger.
type Detector struct {
	sensors map[int]*sensor
}

type sensor struct {
	direction light.Direction
	occupied  bool
}

// New erwartet je Zufahrt den Pin des Kontakts an der Haltelinie.
func New(pins [light.DirectionCount]int) (*Detector, error) {
	d := &Detector{sensors: make(map[int]*sensor, light.DirectionCount)}
	for direction, pin := range pins {
		if _, taken := d.sensors[pin]; taken {
			return nil, fmt.Errorf("BCM %d ist doppelt zugeordnet", pin)
		}
		d.sensors[pin] = &sensor{direction: light.Direction(direction)}
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
	return []SensorEvent{{Direction: s.direction, Occupied: occupied, At: at}}, nil
}

// Reset vergisst alle Pegel. Danach gilt jede Haltelinie als frei.
func (d *Detector) Reset() {
	for _, s := range d.sensors {
		s.occupied = false
	}
}

// Knows sagt, ob dieser Pin zu einer Zufahrt gehoert. Die Schalter gehoeren nicht dazu.
func (d *Detector) Knows(pin int) bool {
	_, ok := d.sensors[pin]
	return ok
}
