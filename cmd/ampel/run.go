package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"ampel/internal/clock"
	"ampel/internal/config"
	"ampel/internal/controller"
	"ampel/internal/display"
	"ampel/internal/hal"
	"ampel/internal/logging"
	"ampel/internal/strategy"
)

// inputBuffer fasst die Flanken, die zwischen zwei Durchlaeufen der Ereignisschleife anfallen.
const inputBuffer = 256

// runControl haelt die Kreuzung in Betrieb, bis der Kontext endet. logDir ueberschreibt das
// Logverzeichnis aus der Konfiguration, wenn es gesetzt ist.
func runControl(ctx context.Context, cfg *config.Config, logDir, mode string, out io.Writer) error {
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

	active, err := newStrategy(cfg, mode)
	if err != nil {
		return err
	}
	power := newPower(cfg, inputs)

	if logDir == "" {
		logDir = cfg.Logging.Dir
	}
	started := time.Now()
	run, err := logging.NewRun(logDir, started, cfg.Logging.Buffer, active.Name())
	if err != nil {
		return err
	}
	defer func() { _ = run.Close() }()

	setup, err := cfg.Setup()
	if err != nil {
		return err
	}
	recorder := logging.NewRecorder(run, started, logging.DefaultSettle, active.Name())
	observers := controller.Observers{recorder}

	// Die Anzeige ist Zubehoer. Laesst sie sich nicht oeffnen, steuert die Kreuzung trotzdem.
	screen, closeDisplay, err := openDisplay(chip, cfg, out)
	if err != nil {
		fmt.Fprintln(out, "Hinweis: Anzeige nicht verfuegbar:", err)
	} else {
		defer closeDisplay()
	}
	var panelDisplay *display.Observer
	if screen != nil {
		panelDisplay = display.NewObserver(screen, nil,
			func(err error) { fmt.Fprintln(out, "Anzeige:", err) })
		observers = append(observers, panelDisplay)
	}

	setup.Strategy = active
	setup.Power = power
	setup.Watchdog = controller.DefaultWatchdog
	setup.Clock = clock.NewReal()
	setup.Writer = driver
	setup.Inputs = pump(ctx, inputs.Events())
	setup.Observer = observers

	control, err := controller.Build(setup)
	if err != nil {
		return err
	}

	if panelDisplay != nil {
		panelDisplay.Source(control.Snapshot)
		panelDisplay.Sample(started, control.Snapshot(started))
	}
	recorder.Start(started)
	fmt.Fprintf(out, "Betrieb gestartet, Modus %s, Lauf %s, Logverzeichnis %s\n", control.Mode(), run.ID(), logDir)
	err = control.Run(ctx)
	recorder.Stop(time.Now())

	fmt.Fprintf(out, "Beendet. %d Fahrzeuge, mittlere Wartezeit %s\n",
		control.Metrics().Total(), control.Metrics().MeanAll())
	fmt.Fprintf(out, "%d Flanken und %d Logzeilen verworfen\n", inputs.Dropped(), run.Dropped())
	if logErr := recorder.Err(); logErr != nil {
		fmt.Fprintln(out, "Hinweis:", logErr)
	}
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}

// newStrategy waehlt die Regelung. Der Betrieb ist adaptiv; die Festzeit dient nur der
// Vergleichsmessung und wird auf der Kommandozeile angefordert.
func newStrategy(cfg *config.Config, mode string) (strategy.Strategy, error) {
	switch mode {
	case "adaptiv":
		return cfg.Following()
	case "festzeit":
		return strategy.NewFixed(cfg.Fixed.Green.Duration()), nil
	}
	return nil, fmt.Errorf("unbekannte Betriebsart %q, erlaubt sind adaptiv und festzeit", mode)
}

// newPower verdrahtet den Hauptschalter. Laesst sich der Pegel nicht lesen, gilt die Anlage
// als eingeschaltet: eine dunkle Kreuzung waere der schlechtere Startzustand.
func newPower(cfg *config.Config, source hal.InputSource) *controller.Power {
	on := true
	if closed, err := source.Read(cfg.Hardware.PowerSwitch); err == nil {
		on = closed
	}
	return &controller.Power{Pin: cfg.Hardware.PowerSwitch, On: on}
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
