// Paket config beschreibt die Konfiguration der Ampelsteuerung und laedt sie aus YAML.
package config

import "time"

type Config struct {
	Hardware     Hardware    `yaml:"hardware"`
	Timing       Timing      `yaml:"timing"`
	Fixed        Fixed       `yaml:"fixed"`
	Adaptive     Adaptive    `yaml:"adaptive"`
	QueueMapping map[int]int `yaml:"queue_mapping"`
	Logging      Logging     `yaml:"logging"`
	Learning     Learning    `yaml:"learning"`
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
	MinGreen  Millis `yaml:"min_green_ms"`
	MaxGreen  Millis `yaml:"max_green_ms"`
	Cycle     Millis `yaml:"cycle_ms"`
	Gap       Millis `yaml:"gap_ms"`
	Extension Millis `yaml:"extension_ms"`
	MaxWait   Millis `yaml:"max_wait_ms"`
}

type Fixed struct {
	Green Millis `yaml:"green_ms"`
}

type Adaptive struct {
	DemandAlpha float64 `yaml:"demand_alpha"`
	LearnAlpha  float64 `yaml:"learn_alpha"`
	BlendK      float64 `yaml:"blend_k"`
}

type Logging struct {
	Dir           string `yaml:"dir"`
	StateInterval Millis `yaml:"state_interval_ms"`
	Buffer        int    `yaml:"buffer"`
}

type Learning struct {
	Path         string `yaml:"path"`
	SaveInterval Millis `yaml:"save_interval_ms"`
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

// CycleEffective ist die Umlaufzeit abzueglich der Zwischenzeiten beider Wechsel eines Umlaufs.
// Nur diese Zeit steht als Gruenzeit zur Verteilung bereit.
func (t Timing) CycleEffective() time.Duration {
	return t.Cycle.Duration() - 2*t.Intergreen()
}
