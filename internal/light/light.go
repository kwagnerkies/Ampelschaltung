package light

import (
	"fmt"
	"slices"
)

type Direction int

const (
	North Direction = iota
	East
	South
	West
)

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

type Aspect uint8

const (
	AspectRed Aspect = iota
	AspectRedYellow
	AspectGreen
	AspectYellow
	AspectOff
	AspectYellowFlash
)

type Lamps struct {
	Red    bool
	Yellow bool
	Green  bool
}

func (a Aspect) Lamps() Lamps {
	switch a {
	case AspectRed:
		return Lamps{Red: true}
	case AspectRedYellow:
		return Lamps{Red: true, Yellow: true}
	case AspectGreen:
		return Lamps{Green: true}
	case AspectYellow, AspectYellowFlash:
		return Lamps{Yellow: true}
	}
	return Lamps{}
}

var successors = map[Aspect][]Aspect{
	AspectRed:         {AspectRedYellow},
	AspectRedYellow:   {AspectGreen, AspectRed},
	AspectGreen:       {AspectYellow},
	AspectYellow:      {AspectRed},
	AspectOff:         {AspectRed},
	AspectYellowFlash: {AspectRed},
}

func (a Aspect) CanFollow(previous Aspect) bool {
	if a == previous || a == AspectYellowFlash || a == AspectOff {
		return true
	}
	return slices.Contains(successors[previous], a)
}

func (a Aspect) Releasing() bool {
	switch a {
	case AspectRedYellow, AspectGreen, AspectYellow:
		return true
	}
	return false
}

func (a Aspect) String() string {
	switch a {
	case AspectRed:
		return "Rot"
	case AspectRedYellow:
		return "RotGelb"
	case AspectGreen:
		return "Gruen"
	case AspectYellow:
		return "Gelb"
	case AspectOff:
		return "Aus"
	case AspectYellowFlash:
		return "GelbBlinken"
	}
	return "unbekannt"
}

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

func (h *Head) Set(a Aspect) error {
	if !a.CanFollow(h.aspect) {
		return fmt.Errorf("zufahrt %s: %s darf nicht auf %s folgen", h.direction, a, h.aspect)
	}
	h.aspect = a
	return nil
}

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
