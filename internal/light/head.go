package light

import "fmt"

// Head ist ein Ampelkopf aus drei Lampen. Er beginnt abgeschaltet, weil die Hardware beim
// Prozessstart dunkel ist.
type Head struct {
	direction Direction
	aspect    Aspect
}

func NewHead(d Direction) *Head {
	return &Head{direction: d, aspect: AspectOff}
}

func (h *Head) Direction() Direction { return h.direction }

func (h *Head) Aspect() Aspect { return h.aspect }

func (h *Head) Lamps() Lamps { return h.aspect.Lamps() }

// Set uebernimmt ein Signalbild, wenn es der deutschen Signalfolge entspricht. Ein Sprung
// von Gruen auf Rot ohne Gelb wird abgewiesen.
func (h *Head) Set(a Aspect) error {
	if !a.CanFollow(h.aspect) {
		return fmt.Errorf("zufahrt %s: %s darf nicht auf %s folgen", h.direction, a, h.aspect)
	}
	h.aspect = a
	return nil
}

// Heads sind die vier Koepfe der Kreuzung in der Reihenfolge Nord, Ost, Sued, West.
type Heads [DirectionCount]*Head

func NewHeads() Heads {
	var heads Heads
	for i, d := range Directions() {
		heads[i] = NewHead(d)
	}
	return heads
}

func (h Heads) Aspects() [DirectionCount]Aspect {
	var aspects [DirectionCount]Aspect
	for i, head := range h {
		aspects[i] = head.aspect
	}
	return aspects
}

// Set schaltet mehrere Koepfe gemeinsam. Schlaegt ein Kopf fehl, bleibt keiner veraendert,
// damit nie ein halb geschalteter Zustand entsteht.
func (h Heads) Set(aspects [DirectionCount]Aspect) error {
	previous := h.Aspects()
	for i, aspect := range aspects {
		if err := h[i].Set(aspect); err != nil {
			for j, old := range previous {
				h[j].aspect = old
			}
			return err
		}
	}
	return nil
}
