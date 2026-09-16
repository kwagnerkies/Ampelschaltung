package detector

import (
	"fmt"
	"time"

	"ampel/internal/light"
)

// Detector ordnet GPIO-Pins den Zufahrten zu und entprellt als zweite Stufe hinter der
// Entprellung im Kernel. Ein Pegel gilt erst als uebernommen, wenn er die Ruhezeit lang
// stabil war. Ein prellender Kontakt erzeugt damit kein Ereignis.
type Detector struct {
	sensors  map[int]*sensor
	order    []int
	debounce time.Duration
}

type sensor struct {
	direction light.Direction
	index     int
	accepted  bool
	raw       bool
	rawAt     time.Time
}

// New erwartet die Sensorpins je Zufahrt in der Reihenfolge Haltelinie, dann aufwaerts.
func New(pins [light.DirectionCount][]int, debounce time.Duration) (*Detector, error) {
	d := &Detector{
		sensors:  make(map[int]*sensor),
		debounce: debounce,
	}
	for direction, approach := range pins {
		for index, pin := range approach {
			if _, taken := d.sensors[pin]; taken {
				return nil, fmt.Errorf("BCM %d ist doppelt zugeordnet", pin)
			}
			d.sensors[pin] = &sensor{direction: light.Direction(direction), index: index}
			d.order = append(d.order, pin)
		}
	}
	if len(d.sensors) == 0 {
		return nil, fmt.Errorf("kein sensor zugeordnet")
	}
	return d, nil
}

// Feed nimmt eine Flanke auf. Zurueck kommen die Ereignisse, die dadurch gueltig werden.
func (d *Detector) Feed(pin int, occupied bool, at time.Time) ([]SensorEvent, error) {
	s, ok := d.sensors[pin]
	if !ok {
		return nil, fmt.Errorf("BCM %d gehoert zu keiner zufahrt", pin)
	}
	if occupied != s.raw {
		s.raw = occupied
		s.rawAt = at
	}
	return d.settle(at), nil
}

// Tick uebernimmt Pegel, die inzwischen lange genug stabil sind. Der Regelkreis ruft das im
// Takt seiner Schleife auf, damit eine Flanke am Ende eines Prellens nicht liegen bleibt.
func (d *Detector) Tick(now time.Time) []SensorEvent {
	return d.settle(now)
}

// Reset vergisst alle Pegel. Nach dem Reset gilt jede Zufahrt als frei.
func (d *Detector) Reset() {
	for _, s := range d.sensors {
		s.accepted = false
		s.raw = false
		s.rawAt = time.Time{}
	}
}

func (d *Detector) settle(now time.Time) []SensorEvent {
	var events []SensorEvent
	for _, pin := range d.order {
		s := d.sensors[pin]
		if s.raw == s.accepted || now.Sub(s.rawAt) < d.debounce {
			continue
		}
		s.accepted = s.raw
		events = append(events, SensorEvent{
			Direction: s.direction,
			Index:     s.index,
			Occupied:  s.accepted,
			At:        s.rawAt,
		})
	}
	return events
}

// Knows sagt, ob dieser Pin zu einem Sensor gehoert. Kippschalter und Taster gehoeren nicht
// dazu und werden vom Regelkreis anders behandelt.
func (d *Detector) Knows(pin int) bool {
	_, ok := d.sensors[pin]
	return ok
}
