// Paket config beschreibt die Konfiguration der Ampelsteuerung und laedt sie aus YAML.
package config

import "time"

type Config struct {
	Hardware     Hardware    `yaml:"hardware"`
	Timing       Timing      `yaml:"timing"`
	Fixed        Fixed       `yaml:"fixed"`
	QueueMapping map[int]int `yaml:"queue_mapping"`
	Logging      Logging     `yaml:"logging"`
	Display      Display     `yaml:"display"`
}

type Hardware struct {
	Chip          string        `yaml:"chip"`
	ShiftRegister ShiftRegister `yaml:"shift_register"`
	Sensors       Sensors       `yaml:"sensors"`
	ModeSwitch    int           `yaml:"mode_switch"`
	ResetButton   int           `yaml:"reset_button"`
	Debounce      Millis        `yaml:"debounce_ms"`
}

type ShiftRegister struct {
	Data     int      `yaml:"data"`
	Clock    int      `yaml:"clock"`
	Latch    int      `yaml:"latch"`
	BitOrder []string `yaml:"bit_order"`
}

type Sensors struct {
	North []int `yaml:"north"`
	East  []int `yaml:"east"`
	South []int `yaml:"south"`
	West  []int `yaml:"west"`
}

type Timing struct {
	Yellow    Millis `yaml:"yellow_ms"`
	RedYellow Millis `yaml:"red_yellow_ms"`
	AllRed    Millis `yaml:"all_red_ms"`
	// BaseGreen ist die Grundgruenzeit der adaptiven Steuerung, MaxGreen ihre Obergrenze.
	BaseGreen Millis `yaml:"base_green_ms"`
	MaxGreen  Millis `yaml:"max_green_ms"`
	// Follow ist der groesste Abstand, in dem ein Fahrzeug noch als dicht folgend gilt,
	// Extension die Verlaengerung, die es ausloest.
	Follow    Millis `yaml:"follow_ms"`
	Extension Millis `yaml:"extension_ms"`
}

type Fixed struct {
	Green Millis `yaml:"green_ms"`
}

// Display ist die Anzeige der Gruenzeiten. Ohne sie laeuft die Kreuzung weiter, deshalb ist
// sie abschaltbar.
type Display struct {
	Enabled  bool   `yaml:"enabled"`
	Device   string `yaml:"spi"`
	SpeedHz  int    `yaml:"speed_hz"`
	DC       int    `yaml:"dc"`
	Reset    int    `yaml:"reset"`
	Rotation string `yaml:"rotation"`
}

type Logging struct {
	Dir           string `yaml:"dir"`
	StateInterval Millis `yaml:"state_interval_ms"`
	Buffer        int    `yaml:"buffer"`
}

// Approaches liefert die Sensorpins in der festen Reihenfolge Nord, Ost, Sued, West.
func (s Sensors) Approaches() [4][]int {
	return [4][]int{s.North, s.East, s.South, s.West}
}

// SensorCount ist die Anzahl Sensoren je Zufahrt. Die Validierung stellt sicher, dass alle
// Zufahrten gleich viele haben.
func (s Sensors) SensorCount() int {
	return len(s.North)
}

// Intergreen ist die Summe der Zwischenzeiten eines Phasenwechsels: Gelb, Allrot, RotGelb.
func (t Timing) Intergreen() time.Duration {
	return t.Yellow.Duration() + t.AllRed.Duration() + t.RedYellow.Duration()
}
