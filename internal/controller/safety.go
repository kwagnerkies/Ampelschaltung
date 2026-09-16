package controller

import (
	"errors"
	"fmt"

	"ampel/internal/light"
)

// conflicts ist die Konfliktmatrix der Kreuzung. Nord und Sued sowie Ost und West sind
// gegenueberliegende Zufahrten und vertraeglich, jedes andere Paar kreuzt sich.
var conflicts = [light.DirectionCount][light.DirectionCount]bool{
	light.North: {light.East: true, light.West: true},
	light.East:  {light.North: true, light.South: true},
	light.South: {light.East: true, light.West: true},
	light.West:  {light.North: true, light.South: true},
}

// ErrConflict meldet einen unzulaessigen Signalzustand.
var ErrConflict = errors.New("unzulaessiger signalzustand")

// Check prueft die Signalbilder aller Koepfe gegen die Konfliktmatrix. Kein Ausgabepfad zur
// Hardware darf diese Funktion umgehen.
func Check(aspects [light.DirectionCount]light.Aspect) error {
	for i, own := range aspects {
		if !own.Releasing() {
			continue
		}
		for j, other := range aspects {
			if i == j {
				continue
			}
			if conflicts[i][j] && other.Releasing() {
				return fmt.Errorf("%w: %s zeigt %s, %s zeigt %s",
					ErrConflict, light.Direction(i), own, light.Direction(j), other)
			}
			// Ein dunkler Kopf neben einer Freigabe bedeutet eine ungeregelte Kreuzung und
			// ist genauso gefaehrlich wie zwei gruene Richtungen.
			if other == light.AspectOff {
				return fmt.Errorf("%w: %s zeigt %s, %s ist dunkel",
					ErrConflict, light.Direction(i), own, light.Direction(j))
			}
		}
	}
	return nil
}

// Conflicting sagt, ob sich zwei Zufahrten kreuzen.
func Conflicting(a, b light.Direction) bool {
	return conflicts[a][b]
}
