package light

import "fmt"

// Bus bildet die Signalbilder aller Koepfe auf das Bitmuster der Schieberegisterkette ab.
// Er schreibt nicht selbst; das Muster geht durch die Sicherheitspruefung an den Treiber.
type Bus struct {
	positions [DirectionCount][3]int
	bits      int
}

// NewBus erwartet die Bitpositionen als [Richtung][Rot, Gelb, Gruen] und die Laenge der Kette.
func NewBus(positions [DirectionCount][3]int, bits int) (*Bus, error) {
	if bits <= 0 {
		return nil, fmt.Errorf("die kette hat %d bits", bits)
	}
	used := make(map[int]bool, DirectionCount*3)
	for i, head := range positions {
		for j, position := range head {
			if position < 0 || position >= bits {
				return nil, fmt.Errorf("position %d fuer %s %s liegt ausserhalb der kette mit %d bits",
					position, Direction(i), colorName(j), bits)
			}
			if used[position] {
				return nil, fmt.Errorf("position %d ist doppelt belegt", position)
			}
			used[position] = true
		}
	}
	return &Bus{positions: positions, bits: bits}, nil
}

func (b *Bus) Bits() int { return b.bits }

// Pattern liefert das Bitmuster fuer die uebergebenen Signalbilder. Index 0 des Musters wird
// zuerst ausgeschoben.
func (b *Bus) Pattern(aspects [DirectionCount]Aspect) []bool {
	pattern := make([]bool, b.bits)
	for i, aspect := range aspects {
		lamps := aspect.Lamps()
		pattern[b.positions[i][0]] = lamps.Red
		pattern[b.positions[i][1]] = lamps.Yellow
		pattern[b.positions[i][2]] = lamps.Green
	}
	return pattern
}

func colorName(index int) string {
	switch index {
	case 0:
		return "Rot"
	case 1:
		return "Gelb"
	case 2:
		return "Gruen"
	}
	return "unbekannt"
}
