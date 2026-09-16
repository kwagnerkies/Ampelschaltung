// Paket hal kapselt die physische Ein- und Ausgabe. Es kennt Pins und Register, aber keine Ampeln.
package hal

import "time"

// LampDriver nimmt das Bitmuster der Lampenkette entgegen. Index 0 ist die erste Position
// der Reihenfolge aus der Konfiguration.
type LampDriver interface {
	Write(pattern []bool) error
	Clear() error
	Close() error
}

// InputEvent ist eine entprellte Flanke an einem Eingang. Active bedeutet geschlossener
// Kontakt; alle Eingaenge schalten gegen Masse, physisch ist das der Low-Pegel.
type InputEvent struct {
	Pin    int
	Active bool
	Time   time.Time
}

// InputSource liefert Flanken aller beobachteten Eingaenge auf einem Kanal. Read dient den
// Eingaengen, die zyklisch abgefragt werden statt auf Flanken zu reagieren.
type InputSource interface {
	Events() <-chan InputEvent
	Read(pin int) (bool, error)
	Close() error
}

// OutputLine ist eine einzelne digitale Ausgangsleitung.
type OutputLine interface {
	Set(high bool) error
	Close() error
}
