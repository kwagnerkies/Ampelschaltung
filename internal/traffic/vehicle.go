// Paket traffic fuehrt den Zustand der Zufahrten und die Kennzahlen der Wartezeiten.
package traffic

import (
	"time"

	"ampel/internal/light"
)

// Arrival ist ein Fahrzeug, das die Erfassung erreicht hat. Phase wird als Zahl mitgefuehrt,
// damit die Zufahrt den Phasenautomaten nicht kennen muss.
type Arrival struct {
	At    time.Time
	Queue int
	Phase int
}

// Departure ist ein Fahrzeug, das die Haltelinie ueberfahren hat. Wait ist die Zeit von der
// Ankunft in der Erfassung bis zur Ueberfahrt, also die Verlustzeit an dieser Zufahrt.
type Departure struct {
	Direction light.Direction
	Arrival   Arrival
	At        time.Time
	Wait      time.Duration
}

// Tracker verfolgt die Fahrzeuge einer Zufahrt in der Reihenfolge ihrer Ankunft. Fahrzeuge
// ueberholen im Modell nicht, deshalb genuegt eine Warteschlange.
type Tracker struct {
	waiting []Arrival
}

func (t *Tracker) Arrive(a Arrival) {
	t.waiting = append(t.waiting, a)
}

// Depart nimmt das aelteste wartende Fahrzeug heraus. Ohne wartendes Fahrzeug ist das
// zweite Ergebnis false; das passiert, wenn ein Fahrzeug von Hand auf die Linie gesetzt wird.
func (t *Tracker) Depart(direction light.Direction, at time.Time) (Departure, bool) {
	if len(t.waiting) == 0 {
		return Departure{}, false
	}
	arrival := t.waiting[0]
	t.waiting = t.waiting[1:]
	return Departure{
		Direction: direction,
		Arrival:   arrival,
		At:        at,
		Wait:      at.Sub(arrival.At),
	}, true
}

func (t *Tracker) Waiting() int { return len(t.waiting) }

func (t *Tracker) Reset() { t.waiting = nil }
