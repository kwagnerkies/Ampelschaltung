package detector

import (
	"ampel/internal/light"
	"fmt"
	"time"
)

type Detector struct {
	sensors map[int]*sensor
}

type sensor struct {
	direction light.Direction
	occupied  bool
}

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

func (d *Detector) Reset() {
	for _, s := range d.sensors {
		s.occupied = false
	}
}

func (d *Detector) Knows(pin int) bool {
	_, ok := d.sensors[pin]
	return ok
}

type SensorEvent struct {
	Direction light.Direction
	Occupied  bool
	At        time.Time
}
