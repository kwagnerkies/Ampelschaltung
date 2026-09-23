// Paket config beschreibt die Konfiguration der Ampelsteuerung und laedt sie aus YAML.
package config

import "time"

type Config struct {
	Hardware Hardware `yaml:"hardware"`
	Timing   Timing   `yaml:"timing"`
	Display  Display  `yaml:"display"`
}

type Hardware struct {
	Chip        string  `yaml:"chip"`
	Lamps       Lamps   `yaml:"lamps"`
	Sensors     Sensors `yaml:"sensors"`
	PowerSwitch int     `yaml:"power_switch"`
	FaultSwitch int     `yaml:"fault_switch"`
	Debounce    Millis  `yaml:"debounce_ms"`
}

// Lamps sind die zwoelf LED-Leitungen, je Ampelkopf Rot, Gelb, Gruen.
type Lamps struct {
	North [3]int `yaml:"north"`
	East  [3]int `yaml:"east"`
	South [3]int `yaml:"south"`
	West  [3]int `yaml:"west"`
}

// Heads liefert die Lampenpins in der festen Reihenfolge Nord, Ost, Sued, West.
func (l Lamps) Heads() [4][3]int {
	return [4][3]int{l.North, l.East, l.South, l.West}
}

// Sensors ist je Zufahrt der Reed-Kontakt an der Haltelinie.
type Sensors struct {
	North int `yaml:"north"`
	East  int `yaml:"east"`
	South int `yaml:"south"`
	West  int `yaml:"west"`
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

// Approaches liefert die Sensorpins in der festen Reihenfolge Nord, Ost, Sued, West.
func (s Sensors) Approaches() [4]int {
	return [4]int{s.North, s.East, s.South, s.West}
}

// Intergreen ist die Summe der Zwischenzeiten eines Phasenwechsels: Gelb, Allrot, RotGelb.
func (t Timing) Intergreen() time.Duration {
	return t.Yellow.Duration() + t.AllRed.Duration() + t.RedYellow.Duration()
}
