package main

import (
	"context"
	"io"
	"testing"

	"ampel/internal/config"
	"ampel/internal/hal"
)

// Der Lampentest muss jede Lampe genau einmal und einzeln zeigen und am Ende alles
// abschalten. Sonst bleibt nach dem Test ein Licht stehen.
func TestWalkLampsLightsEachLampAlone(t *testing.T) {
	cfg := config.Default()
	bits := len(cfg.Hardware.ShiftRegister.BitOrder)
	driver := hal.NewMock(bits, 1)

	if err := walkLamps(context.Background(), io.Discard, driver, &cfg, 0); err != nil {
		t.Fatalf("walkLamps: %v", err)
	}

	history := driver.History()
	lamps := config.LampNames()
	if len(history) != len(lamps)+1 {
		t.Fatalf("%d Schreibzugriffe, erwartet %d", len(history), len(lamps)+1)
	}

	positions := cfg.Hardware.ShiftRegister.LampPositions()
	for i, lamp := range lamps {
		pattern := history[i]
		want := positions[lamp]
		for bit, on := range pattern {
			if on != (bit == want) {
				t.Errorf("bei lampe %s ist bit %d %v, erwartet %v", lamp, bit, on, bit == want)
			}
		}
	}
	for bit, on := range history[len(lamps)] {
		if on {
			t.Errorf("nach dem Test leuchtet noch bit %d", bit)
		}
	}
}

func TestInputPinsCoversSensorsAndControls(t *testing.T) {
	cfg := config.Default()
	pins, labels := inputPins(&cfg)

	if want := 4*cfg.Hardware.Sensors.SensorCount() + 2; len(pins) != want {
		t.Fatalf("%d Eingaenge, erwartet %d", len(pins), want)
	}
	for _, pin := range pins {
		if labels[pin] == "" {
			t.Errorf("BCM %d hat keine Bezeichnung", pin)
		}
	}
	if got := labels[cfg.Hardware.Sensors.North[0]]; got != "Nord Sensor 0" {
		t.Errorf("Bezeichnung %q, erwartet \"Nord Sensor 0\"", got)
	}
}
