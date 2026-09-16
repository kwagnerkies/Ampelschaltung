package main

import (
	"context"
	"errors"
	"fmt"
	"io"

	"ampel/internal/clock"
	"ampel/internal/config"
	"ampel/internal/controller"
	"ampel/internal/display"
	"ampel/internal/hal"
)

// inputBuffer fasst die Flanken, die zwischen zwei Durchlaeufen der Ereignisschleife anfallen.
const inputBuffer = 256

// runControl haelt die Kreuzung in Betrieb, bis der Kontext endet.
func runControl(ctx context.Context, cfg *config.Config, out io.Writer) error {
	chip, err := hal.OpenChip(cfg.Hardware.Chip)
	if err != nil {
		return err
	}
	defer func() { _ = chip.Close() }()

	driver, err := openLamps(chip, cfg)
	if err != nil {
		return err
	}
	defer func() { _ = driver.Close() }()

	pins, _ := inputPins(cfg)
	inputs, err := hal.NewGPIOInput(cfg.Hardware.Chip, pins, cfg.Hardware.Debounce.Duration(), inputBuffer, clock.NewReal())
	if err != nil {
		return err
	}
	defer func() { _ = inputs.Close() }()

	setup, err := cfg.Setup()
	if err != nil {
		return err
	}
	setup.Strategy, err = cfg.Following()
	if err != nil {
		return err
	}
	setup.Switches = readSwitches(cfg, inputs)
	setup.Watchdog = controller.DefaultWatchdog
	setup.Clock = clock.NewReal()
	setup.Writer = driver
	setup.Inputs = pump(ctx, inputs.Events())

	// Die Anzeige ist Zubehoer. Laesst sie sich nicht oeffnen, steuert die Kreuzung trotzdem.
	screen, closeDisplay, err := openDisplay(chip, cfg, out)
	if err != nil {
		fmt.Fprintln(out, "Hinweis: Anzeige nicht verfuegbar:", err)
	} else {
		defer closeDisplay()
	}
	var panel *display.Observer
	if screen != nil {
		panel = display.NewObserver(screen, nil, func(err error) { fmt.Fprintln(out, "Anzeige:", err) })
		setup.Observer = panel
	}

	control, err := controller.Build(setup)
	if err != nil {
		return err
	}
	if panel != nil {
		panel.Source(control.Snapshot)
	}

	fmt.Fprintln(out, "Betrieb gestartet, Verlaengerung bei dicht folgenden Fahrzeugen")
	err = control.Run(ctx)
	fmt.Fprintf(out, "Beendet. %d Flanken verworfen\n", inputs.Dropped())
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}

// readSwitches liest die Pegel beim Start. Laesst sich ein Schalter nicht lesen, gilt die
// Anlage als eingeschaltet und nicht gestoert: eine dunkle Kreuzung waere der schlechtere
// Startzustand.
func readSwitches(cfg *config.Config, source hal.InputSource) *controller.Switches {
	s := &controller.Switches{
		PowerPin: cfg.Hardware.PowerSwitch,
		FaultPin: cfg.Hardware.FaultSwitch,
		PowerOn:  true,
	}
	if closed, err := source.Read(s.PowerPin); err == nil {
		s.PowerOn = closed
	}
	if closed, err := source.Read(s.FaultPin); err == nil {
		s.FaultOn = closed
	}
	return s
}

// pump uebersetzt die Flanken der Hardwareschicht in die Ereignisse des Regelkreises. Damit
// bleibt der Regelkreis frei von jeder Kenntnis der Hardware.
func pump(ctx context.Context, events <-chan hal.InputEvent) <-chan controller.Input {
	inputs := make(chan controller.Input, inputBuffer)
	go func() {
		defer close(inputs)
		for {
			select {
			case <-ctx.Done():
				return
			case event := <-events:
				select {
				case inputs <- controller.Input{Pin: event.Pin, Active: event.Active, Time: event.Time}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return inputs
}
