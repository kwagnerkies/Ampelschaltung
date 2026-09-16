package light

// Direction benennt eine Zufahrt. Die Reihenfolge ist im ganzen Projekt gleich und wird von
// Konfiguration, Sensoren und Lampenbus gleichermassen verwendet.
type Direction int

const (
	North Direction = iota
	East
	South
	West
)

// DirectionCount ist die Anzahl der Zufahrten der Kreuzung.
const DirectionCount = 4

func Directions() [DirectionCount]Direction {
	return [DirectionCount]Direction{North, East, South, West}
}

func (d Direction) String() string {
	switch d {
	case North:
		return "Nord"
	case East:
		return "Ost"
	case South:
		return "Sued"
	case West:
		return "West"
	}
	return "unbekannt"
}

func (d Direction) valid() bool {
	return d >= North && d <= West
}
