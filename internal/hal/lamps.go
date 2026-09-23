package hal

import "fmt"

// Lamps treibt die Ampel-LEDs unmittelbar an je einer GPIO-Leitung. Es leuchten nie alle
// zwoelf gleichzeitig: im ungeguenstigsten Fall zeigen zwei Koepfe Rot mit Gelb und zwei Rot,
// also sechs Lampen. Bei wenigen Milliampere je LED bleibt das im Budget des Pi.
type Lamps struct {
	lines []OutputLine
}

var _ LampDriver = (*Lamps)(nil)

func NewLamps(lines []OutputLine) *Lamps { return &Lamps{lines: lines} }

func (l *Lamps) Write(state []bool) error {
	if len(state) != len(l.lines) {
		return fmt.Errorf("%d lampenzustaende, die kreuzung hat %d leitungen", len(state), len(l.lines))
	}
	for i, on := range state {
		if err := l.lines[i].Set(on); err != nil {
			return fmt.Errorf("lampe %d schalten: %w", i, err)
		}
	}
	return nil
}

func (l *Lamps) Clear() error { return l.Write(make([]bool, len(l.lines))) }

func (l *Lamps) Close() error {
	var first error
	for _, line := range l.lines {
		if err := line.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}
