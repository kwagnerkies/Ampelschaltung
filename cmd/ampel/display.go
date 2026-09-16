package main

import (
	"fmt"
	"io"

	"ampel/internal/config"
	"ampel/internal/display"
	"ampel/internal/hal"
)

// canvas verbindet den Displaytreiber mit der Darstellung. Die Hardwareschicht kennt die
// Darstellung nicht und umgekehrt, deshalb steht die Uebersetzung hier.
type canvas struct {
	tft *hal.TFT
}

func (c canvas) Size() (int, int) { return c.tft.Size() }

func (c canvas) Fill(x, y, width, height int, color display.Color) error {
	return c.tft.Fill(x, y, width, height, uint16(color))
}

// openDisplay richtet die Anzeige ein. Sie ist Zubehoer: faellt sie aus, laeuft die Kreuzung
// weiter und der Grund steht im Log.
func openDisplay(chip *hal.Chip, cfg *config.Config, out io.Writer) (*display.Screen, func(), error) {
	if !cfg.Display.Enabled {
		return nil, func() {}, nil
	}
	bus, err := hal.OpenSPI(cfg.Display.Device, cfg.Display.SpeedHz)
	if err != nil {
		return nil, nil, err
	}
	dc, err := chip.Output(cfg.Display.DC)
	if err != nil {
		_ = bus.Close()
		return nil, nil, fmt.Errorf("anzeige, dc-leitung: %w", err)
	}
	var reset hal.OutputLine
	if cfg.Display.Reset >= 0 {
		reset, err = chip.Output(cfg.Display.Reset)
		if err != nil {
			_ = dc.Close()
			_ = bus.Close()
			return nil, nil, fmt.Errorf("anzeige, reset-leitung: %w", err)
		}
	}
	tft, err := hal.NewTFT(bus, dc, reset, cfg.Display.TFTRotation())
	if err != nil {
		_ = dc.Close()
		_ = bus.Close()
		return nil, nil, err
	}
	closer := func() {
		_ = tft.Close()
		_ = dc.Close()
		if reset != nil {
			_ = reset.Close()
		}
	}
	fmt.Fprintf(out, "Anzeige an %s, Aufloesung %s\n", cfg.Display.Device, size(tft))
	return display.New(canvas{tft: tft}), closer, nil
}

// showTestPattern zeigt alle vier Felder mit 88 in ihren Farben. Damit pruefst du am Aufbau
// Verdrahtung, Drehung und Farbreihenfolge in einem Blick: steht 88 auf dem Kopf, stimmt die
// Drehung nicht, ist Rot blau, sind die Farbkanaele vertauscht.
func showTestPattern(screen *display.Screen) error {
	return screen.Update([4]display.Field{
		{Seconds: 88, Color: display.Green},
		{Seconds: 88, Color: display.Red},
		{Seconds: 88, Color: display.Yellow},
		{Seconds: 88, Color: display.White},
	})
}

func size(tft *hal.TFT) string {
	width, height := tft.Size()
	return fmt.Sprintf("%dx%d", width, height)
}
