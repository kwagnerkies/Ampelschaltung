package main

import (
	"context"
	"io"
	"testing"

	"ampel/internal/config"
	"ampel/internal/controller"
	"ampel/internal/hal"
)

// Der Lampentest muss jede Lampe genau einmal und einzeln zeigen und am Ende alles
// abschalten. Sonst bleibt nach dem Test ein Licht stehen.
func TestWalkLampsLightsEachLampAlone(t *testing.T) {
	cfg := config.Default()
	driver := hal.NewMock(controller.LampCount, 1)

	if err := walkLamps(context.Background(), io.Discard, driver, &cfg, 0); err != nil {
		t.Fatalf("walkLamps: %v", err)
	}

	history := driver.History()
	if len(history) != controller.LampCount+1 {
		t.Fatalf("%d Schreibzugriffe, erwartet %d", len(history), controller.LampCount+1)
	}
	for lamp := 0; lamp < controller.LampCount; lamp++ {
		for index, on := range history[lamp] {
			if on != (index == lamp) {
				t.Errorf("bei lampe %d ist %d %v, erwartet %v", lamp, index, on, index == lamp)
			}
		}
	}
	for index, on := range history[controller.LampCount] {
		if on {
			t.Errorf("nach dem Test leuchtet noch Lampe %d", index)
		}
	}
}

// Die Eingaenge sind die vier Haltelinien und die beiden Schalter, jeder mit Bezeichnung.
func TestInputPinsCoversSensorsAndSwitches(t *testing.T) {
	cfg := config.Default()
	pins, labels := inputPins(&cfg)

	if len(pins) != 6 {
		t.Fatalf("%d Eingaenge, erwartet 6", len(pins))
	}
	for _, pin := range pins {
		if labels[pin] == "" {
			t.Errorf("BCM %d hat keine Bezeichnung", pin)
		}
	}
	if got := labels[cfg.Hardware.Sensors.North]; got != "Nord Haltelinie" {
		t.Errorf("Bezeichnung %q, erwartet \"Nord Haltelinie\"", got)
	}
	if got := labels[cfg.Hardware.FaultSwitch]; got != "Notschalter" {
		t.Errorf("Bezeichnung %q, erwartet \"Notschalter\"", got)
	}
}
