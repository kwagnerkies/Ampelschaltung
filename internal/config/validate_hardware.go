package config

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// maxBCM ist die hoechste BCM-Nummer, die auf der 40-poligen Stiftleiste des Pi liegt.
const maxBCM = 27

// registerBits ist die Laenge der Kette aus zwei kaskadierten 74HC595.
const registerBits = 16

var approachNames = [4]string{"north", "east", "south", "west"}

func (h Hardware) validate() []error {
	var errs []error
	if h.Chip == "" {
		errs = append(errs, errors.New("hardware.chip darf nicht leer sein"))
	}
	if h.Debounce < 0 {
		errs = append(errs, errors.New("hardware.debounce_ms darf nicht negativ sein"))
	}
	errs = append(errs, h.validatePins()...)
	errs = append(errs, h.validateSensorCounts()...)
	errs = append(errs, validateBitOrder(h.ShiftRegister.BitOrder)...)
	return errs
}

func (h Hardware) validatePins() []error {
	var errs []error
	used := make(map[int]string, registerBits)
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

	claim(h.ShiftRegister.Data, "hardware.shift_register.data")
	claim(h.ShiftRegister.Clock, "hardware.shift_register.clock")
	claim(h.ShiftRegister.Latch, "hardware.shift_register.latch")
	for i, pins := range h.Sensors.Approaches() {
		for j, pin := range pins {
			claim(pin, fmt.Sprintf("hardware.sensors.%s[%d]", approachNames[i], j))
		}
	}
	claim(h.PowerSwitch, "hardware.power_switch")
	return errs
}

func (h Hardware) validateSensorCounts() []error {
	var errs []error
	count := h.Sensors.SensorCount()
	if count == 0 {
		errs = append(errs, errors.New("hardware.sensors.north braucht mindestens einen Sensor"))
	}
	for i, pins := range h.Sensors.Approaches() {
		if len(pins) != count {
			errs = append(errs, fmt.Errorf("hardware.sensors.%s hat %d Sensoren, hardware.sensors.north hat %d; alle Zufahrten muessen gleich viele haben", approachNames[i], len(pins), count))
		}
	}
	return errs
}

func validateBitOrder(order []string) []error {
	var errs []error
	if len(order) != registerBits {
		errs = append(errs, fmt.Errorf("hardware.shift_register.bit_order braucht genau %d Eintraege, hat %d", registerBits, len(order)))
	}
	seen := make(map[string]int, len(order))
	for i, name := range order {
		if isFreeBit(name) {
			continue
		}
		if previous, taken := seen[name]; taken {
			errs = append(errs, fmt.Errorf("hardware.shift_register.bit_order nennt %s zweimal, auf Position %d und %d", name, previous, i))
			continue
		}
		seen[name] = i
	}
	var missing []string
	for _, want := range lampBits {
		if _, ok := seen[want]; !ok {
			missing = append(missing, want)
		}
	}
	if len(missing) > 0 {
		slices.Sort(missing)
		errs = append(errs, fmt.Errorf("hardware.shift_register.bit_order fehlen die Lampen: %s", strings.Join(missing, ", ")))
	}
	for name := range seen {
		if !isLampBit(name) {
			errs = append(errs, fmt.Errorf("hardware.shift_register.bit_order kennt %q nicht; erlaubt sind die zwoelf Lampen und freie Positionen", name))
		}
	}
	return errs
}
