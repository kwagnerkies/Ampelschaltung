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
	"ampel/internal/hal"
	"ampel/internal/learning"
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

	panel, err := newPanel(cfg, inputs, mode)
	if err != nil {
		return err
	}
	active := panel.Fixed
	if panel.SwitchClosed {
		active = panel.Adaptive
	}

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
	setup.Strategy = active
	setup.Panel = panel
	setup.Watchdog = controller.DefaultWatchdog
	// Der Lernzustand steht immer bereit, denn der Kippschalter kann jederzeit in den
	// adaptiven Betrieb wechseln. Gelernt wird nur dort, das entscheidet der Regelkreis.
	blender, err := newBlender(cfg, started, out)
	if err != nil {
		return err
	}
	setup.Learner = blender
	setup.Clock = clock.NewReal()
	setup.Writer = driver
	setup.Inputs = pump(ctx, inputs.Events())
	setup.Observer = recorder

	control, err := controller.Build(setup)
	if err != nil {
		return err
	}

	defer blender.Flush()
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

// newBlender laedt den Lernzustand und richtet die Mischung ein. Ein fehlender oder
// unbrauchbarer Stand ist kein Startfehler, sondern ein leerer Anfang mit Hinweis im Log.
func newBlender(cfg *config.Config, started time.Time, out io.Writer) (*learning.Blender, error) {
	path := cfg.Learning.Path
	histogram, err := learning.Load(path)
	if err != nil {
		fmt.Fprintf(out, "Hinweis: %v\n", err)
	}
	return learning.NewBlender(learning.Options{
		Histogram: histogram,
		Alpha:     cfg.Adaptive.LearnAlpha,
		K:         cfg.Adaptive.BlendK,
		SaveEvery: cfg.Learning.SaveInterval.Duration(),
		Now:       started,
		Save:      func(h *learning.Histogram) error { return learning.Save(path, h) },
		OnError:   func(err error) { fmt.Fprintf(out, "Lernzustand nicht gespeichert: %v\n", err) },
	})
}

// newPanel verdrahtet Kippschalter und Reset-Taster. Der beim Start gelesene Schalterpegel
// bestimmt die erste Betriebsart; laesst er sich nicht lesen, gilt die Kommandozeile.
func newPanel(cfg *config.Config, source hal.InputSource, mode string) (*controller.Panel, error) {
	if mode != "festzeit" && mode != "adaptiv" {
		return nil, fmt.Errorf("unbekannte Betriebsart %q, erlaubt sind festzeit und adaptiv", mode)
	}
	adaptive, err := strategy.NewAdaptive(cfg.StrategyParams())
	if err != nil {
		return nil, err
	}
	panel := &controller.Panel{
		SwitchPin:    cfg.Hardware.ModeSwitch,
		ResetPin:     cfg.Hardware.ResetButton,
		Fixed:        strategy.NewFixed(cfg.Fixed.Green.Duration()),
		Adaptive:     adaptive,
		SwitchClosed: mode == "adaptiv",
	}
	if closed, err := source.Read(cfg.Hardware.ModeSwitch); err == nil {
		panel.SwitchClosed = closed
	}
	return panel, nil
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
