package config

import (
	"fmt"
	"slices"
)

// lampPrefixes und lampColors sind die einzige Quelle der Lampennamen. Die Reihenfolge legt
// fest, wie LampNames und LampMatrix zaehlen: Nord, Ost, Sued, West und je Kopf Rot, Gelb, Gruen.
var (
	lampPrefixes = [4]string{"N", "E", "S", "W"}
	lampColors   = [3]string{"red", "yellow", "green"}
	lampBits     = buildLampNames()
)

// LampNames liefert die zwoelf Lampennamen, die die Bitreihenfolge belegen darf.
func LampNames() []string { return slices.Clone(lampBits) }

// LampPositions ordnet jeder Lampe ihre Position in der Registerkette zu. Position 0 wird
// zuerst ausgeschoben und liegt am entferntesten Ausgang.
func (s ShiftRegister) LampPositions() map[string]int {
	positions := make(map[string]int, len(lampBits))
	for i, name := range s.BitOrder {
		if isLampBit(name) {
			positions[name] = i
		}
	}
	return positions
}

// LampMatrix liefert die Bitpositionen als [Richtung][Rot, Gelb, Gruen]. Damit verdrahtet die
// Anwendung den Lampenbus, ohne die Namenskonvention der Konfiguration zu kennen.
func (s ShiftRegister) LampMatrix() ([4][3]int, error) {
	positions := s.LampPositions()
	var matrix [4][3]int
	for i, prefix := range lampPrefixes {
		for j, color := range lampColors {
			name := prefix + "_" + color
			position, ok := positions[name]
			if !ok {
				return matrix, fmt.Errorf("bitreihenfolge nennt %s nicht", name)
			}
			matrix[i][j] = position
		}
	}
	return matrix, nil
}

func buildLampNames() []string {
	names := make([]string, 0, len(lampPrefixes)*len(lampColors))
	for _, prefix := range lampPrefixes {
		for _, color := range lampColors {
			names = append(names, prefix+"_"+color)
		}
	}
	return names
}

func isFreeBit(name string) bool {
	switch name {
	case "", "-", "free", "frei":
		return true
	}
	return false
}

func isLampBit(name string) bool {
	return slices.Contains(lampBits, name)
}
