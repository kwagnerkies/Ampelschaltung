package konfiguration

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("konfiguration lesen: %w", err)
	}
	return Parse(data, path)
}

func Parse(data []byte, name string) (*Config, error) {
	cfg := Default()
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("konfiguration %s: %w", name, err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("konfiguration %s ungueltig:\n%w", name, err)
	}
	return &cfg, nil
}

func (c *Config) Validate() error {
	var errs []error
	errs = append(errs, c.Hardware.validate(c.Display)...)
	errs = append(errs, c.Timing.validate()...)
	errs = append(errs, c.Display.validate()...)
	errs = append(errs, c.API.validate()...)
	return errors.Join(errs...)
}

func (t Timing) validate() []error {
	var errs []error
	durations := []struct {
		name  string
		value Millis
	}{
		{"timing.yellow_ms", t.Yellow},
		{"timing.red_yellow_ms", t.RedYellow},
		{"timing.all_red_ms", t.AllRed},
		{"timing.base_green_ms", t.BaseGreen},
		{"timing.max_green_ms", t.MaxGreen},
		{"timing.follow_ms", t.Follow},
		{"timing.extension_ms", t.Extension},
	}
	for _, d := range durations {
		if d.value <= 0 {
			errs = append(errs, fmt.Errorf("%s muss groesser als null sein", d.name))
		}
	}
	if t.BaseGreen > t.MaxGreen {
		errs = append(errs, fmt.Errorf("timing.base_green_ms (%s) darf timing.max_green_ms (%s) nicht ueberschreiten", t.BaseGreen, t.MaxGreen))
	}
	if t.Extension > t.MaxGreen {
		errs = append(errs, fmt.Errorf("timing.extension_ms (%s) darf timing.max_green_ms (%s) nicht ueberschreiten", t.Extension, t.MaxGreen))
	}
	return errs
}

func (a API) validate() []error {
	if a.Enabled && a.Socket == "" {
		return []error{errors.New("fernbedienung.socket darf nicht leer sein")}
	}
	return nil
}

func (d Display) validate() []error {
	if !d.Enabled {
		return nil
	}
	var errs []error
	if d.Device == "" {
		errs = append(errs, errors.New("display.spi darf nicht leer sein"))
	}
	if d.SpeedHz <= 0 {
		errs = append(errs, errors.New("display.speed_hz muss groesser als null sein"))
	}
	if d.Rotation != "quer" && d.Rotation != "hoch" {
		errs = append(errs, fmt.Errorf("display.rotation ist %q, erlaubt sind quer und hoch", d.Rotation))
	}
	return errs
}

const maxBCM = 27

var approachNames = [4]string{"north", "east", "south", "west"}

var lampNames = [3]string{"rot", "gelb", "gruen"}

var spiPins = map[int]string{
	7:  "spi ce1",
	8:  "spi ce0",
	9:  "spi miso",
	10: "spi mosi",
	11: "spi sclk",
}

func (h Hardware) validate(display Display) []error {
	var errs []error
	if h.Chip == "" {
		errs = append(errs, errors.New("hardware.chip darf nicht leer sein"))
	}
	if h.Debounce < 0 {
		errs = append(errs, errors.New("hardware.debounce_ms darf nicht negativ sein"))
	}
	return append(errs, h.validatePins(display)...)
}

func (h Hardware) validatePins(display Display) []error {
	var errs []error
	used := make(map[int]string, 20)
	claim := func(pin int, name string) {
		if pin < 0 || pin > maxBCM {
			errs = append(errs, fmt.Errorf("%s: BCM %d liegt ausserhalb von 0 bis %d", name, pin, maxBCM))
			return
		}
		if previous, taken := used[pin]; taken {
			errs = append(errs, fmt.Errorf("BCM %d ist doppelt belegt: %s und %s", pin, previous, name))
			return
		}
		used[pin] = name
	}

	for i, head := range h.Lamps.Heads() {
		for j, pin := range head {
			claim(pin, fmt.Sprintf("hardware.lamps.%s.%s", approachNames[i], lampNames[j]))
		}
	}
	for i, pin := range h.Sensors.Approaches() {
		claim(pin, "hardware.sensors."+approachNames[i])
	}
	claim(h.PowerSwitch, "hardware.power_switch")
	claim(h.FaultSwitch, "hardware.fault_switch")
	if display.Enabled {
		claim(display.DC, "display.dc")
		if display.Reset >= 0 {
			claim(display.Reset, "display.reset")
		}
		for pin, name := range spiPins {
			claim(pin, name)
		}
	}
	return errs
}
