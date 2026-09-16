package mode

import "time"

// Button erkennt einen langen Druck. Erst nach hold loest er aus, damit ein versehentlicher
// Tastendruck waehrend der Vorfuehrung den Lernzustand nicht loescht.
type Button struct {
	hold      time.Duration
	pressed   bool
	pressedAt time.Time
	fired     bool
}

func NewButton(hold time.Duration) *Button {
	return &Button{hold: hold}
}

// Poll meldet genau einmal true, sobald der Taster lange genug gedrueckt war. Erst nach dem
// Loslassen ist ein weiteres Ausloesen moeglich.
func (b *Button) Poll(pressed bool, now time.Time) bool {
	if !pressed {
		b.pressed = false
		b.fired = false
		return false
	}
	if !b.pressed {
		b.pressed = true
		b.pressedAt = now
		return false
	}
	if b.fired || now.Sub(b.pressedAt) < b.hold {
		return false
	}
	b.fired = true
	return true
}

// Held ist die Dauer des laufenden Drucks. Sie dient der Anzeige waehrend des Haltens.
func (b *Button) Held(now time.Time) time.Duration {
	if !b.pressed {
		return 0
	}
	return now.Sub(b.pressedAt)
}
