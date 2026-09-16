package detector

import "time"

// Occupancy haelt fest, welche Sensoren einer Zufahrt belegt sind. Ein Reed-Kontakt meldet
// Anwesenheit, nicht Durchfahrt: ein stehendes Fahrzeug haelt ihn dauerhaft geschlossen.
type Occupancy struct {
	occupied []bool
	since    []time.Time
}

func NewOccupancy(sensorCount int) *Occupancy {
	return &Occupancy{
		occupied: make([]bool, sensorCount),
		since:    make([]time.Time, sensorCount),
	}
}

func (o *Occupancy) Count() int { return len(o.occupied) }

func (o *Occupancy) Apply(index int, occupied bool, at time.Time) {
	if index < 0 || index >= len(o.occupied) {
		return
	}
	if o.occupied[index] == occupied {
		return
	}
	o.occupied[index] = occupied
	o.since[index] = at
}

func (o *Occupancy) Occupied(index int) bool {
	if index < 0 || index >= len(o.occupied) {
		return false
	}
	return o.occupied[index]
}

// Since ist der Zeitpunkt der letzten Aenderung an diesem Sensor.
func (o *Occupancy) Since(index int) time.Time {
	if index < 0 || index >= len(o.since) {
		return time.Time{}
	}
	return o.since[index]
}

// Reach ist die Anzahl Sensoren von der Haltelinie bis zum hoechsten belegten Sensor. Sie
// ueberspringt Luecken: ist S2 belegt und S1 frei, reicht der Stau trotzdem bis S2.
func (o *Occupancy) Reach() int {
	reach := 0
	for index, occupied := range o.occupied {
		if occupied {
			reach = index + 1
		}
	}
	return reach
}

// AtStopLine sagt, ob die Haltelinie belegt ist.
func (o *Occupancy) AtStopLine() bool { return o.Occupied(0) }

func (o *Occupancy) Reset() {
	for index := range o.occupied {
		o.occupied[index] = false
		o.since[index] = time.Time{}
	}
}
