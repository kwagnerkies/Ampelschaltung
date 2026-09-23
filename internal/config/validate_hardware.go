package config

import (
	"errors"
	"fmt"
)

// maxBCM ist die hoechste BCM-Nummer, die auf der 40-poligen Stiftleiste des Pi liegt.
const maxBCM = 27

var approachNames = [4]string{"north", "east", "south", "west"}

var lampNames = [3]string{"rot", "gelb", "gruen"}

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

// validatePins prueft jede Leitung einmal: gueltige Nummer und keine Doppelbelegung. Ein
// doppelt vergebener Pin ist der Fehler, der am Aufbau am laengsten unentdeckt bleibt.
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
		claim(display.Reset, "display.reset")
	}
	return errs
}
