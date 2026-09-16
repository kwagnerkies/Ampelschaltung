// Paket mode wertet Kippschalter und Reset-Taster aus.
package mode

import "time"

// Switch entprellt einen Kippschalter durch zyklisches Abfragen. Ein mechanischer Schalter
// prellt laenger als ein Reed-Kontakt, deshalb liegt die Ruhezeit hier bei 100 ms und nicht
// bei den 15 ms der Fahrbahnsensoren.
type Switch struct {
	debounce time.Duration
	level    bool
	raw      bool
	rawAt    time.Time
}

// NewSwitch beginnt mit dem beim Start gelesenen Pegel.
func NewSwitch(initial bool, debounce time.Duration, now time.Time) *Switch {
	return &Switch{debounce: debounce, level: initial, raw: initial, rawAt: now}
}

// Poll nimmt den abgefragten Pegel auf und meldet, ob der uebernommene Pegel gewechselt hat.
func (s *Switch) Poll(level bool, now time.Time) bool {
	if level != s.raw {
		s.raw = level
		s.rawAt = now
		return false
	}
	if s.raw == s.level || now.Sub(s.rawAt) < s.debounce {
		return false
	}
	s.level = s.raw
	return true
}

// Level ist der uebernommene Pegel.
func (s *Switch) Level() bool { return s.level }
