package main

import (
	"ampel/src/clock"
	"ampel/src/konfiguration"
	"ampel/src/signal"
	"ampel/src/steuerung"
	"ampel/src/treiber"
	"ampel/src/treiber/gpio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

const lampDwell = 400 * time.Millisecond

const selftestBuffer = 256

func runSelftest(ctx context.Context, cfg *konfiguration.Config, out io.Writer) error {
	chip, err := gpio.OpenChip(cfg.Hardware.Chip)
	if err != nil {
		return err
	}
	defer func() { _ = chip.Close() }()

	driver, err := openLamps(chip, cfg)
	if err != nil {
		return err
	}
	defer func() { _ = driver.Close() }()

	screen, closeDisplay, err := openDisplay(chip, cfg, out)
	if err != nil {
		fmt.Fprintln(out, "Hinweis: Anzeige nicht verfuegbar:", err)
	} else {
		defer closeDisplay()
	}
	if screen != nil {
		fmt.Fprintln(out, "Anzeigetest: alle vier Felder zeigen 88 in Gruen, Rot, Gelb und Weiss.")
		if err := showTestPattern(screen); err != nil {
			fmt.Fprintln(out, "Anzeige:", err)
		}
	}

	pins, labels := inputPins(cfg)
	inputs, err := gpio.NewGPIOInput(cfg.Hardware.Chip, pins, cfg.Hardware.Debounce.Duration(), selftestBuffer, clock.NewReal())
	if err != nil {
		return err
	}
	defer func() { _ = inputs.Close() }()

	go printEvents(ctx, out, inputs.Events(), labels)

	fmt.Fprintln(out, "Lampentest, jede Lampe leuchtet einzeln:")
	if err := walkLamps(ctx, out, driver, cfg, lampDwell); err != nil {
		return err
	}

	fmt.Fprintln(out, "Signalfolge, beide Freigabephasen einmal:")
	if err := showSequence(ctx, out, steuerung.NewOutput(driver), cfg.Timing); err != nil {
		return err
	}

	fmt.Fprintln(out, "Sensortest, Fahrzeug ueber die Kontakte schieben. Abbruch mit Strg-C.")
	<-ctx.Done()
	fmt.Fprintf(out, "%d Flanken verworfen\n", inputs.Dropped())
	return nil
}

func openLamps(chip *gpio.Chip, cfg *konfiguration.Config) (treiber.LampDriver, error) {
	var lines []treiber.OutputLine
	for i, head := range cfg.Hardware.Lamps.Heads() {
		for j, pin := range head {
			line, err := chip.Output(pin)
			if err != nil {
				for _, opened := range lines {
					_ = opened.Close()
				}
				return nil, fmt.Errorf("lampe %s %s: %w", approachNames[i], lampNames[j], err)
			}
			lines = append(lines, line)
		}
	}
	return gpio.NewLamps(lines), nil
}

func walkLamps(ctx context.Context, out io.Writer, driver treiber.LampDriver, cfg *konfiguration.Config, dwell time.Duration) error {
	for index := 0; index < steuerung.LampCount; index++ {
		pattern := make([]bool, steuerung.LampCount)
		pattern[index] = true
		if err := driver.Write(pattern); err != nil {
			return fmt.Errorf("lampe %d schalten: %w", index, err)
		}
		pin := cfg.Hardware.Lamps.Heads()[index/3][index%3]
		fmt.Fprintf(out, "  BCM %2d  %s %s\n", pin, approachNames[index/3], lampNames[index%3])
		select {
		case <-ctx.Done():
			return driver.Clear()
		case <-time.After(dwell):
		}
	}
	if err := driver.Clear(); err != nil {
		return errors.Join(errors.New("lampen abschalten"), err)
	}
	return nil
}

func inputPins(cfg *konfiguration.Config) ([]int, map[int]string) {
	names := [4]string{"Nord", "Ost", "Sued", "West"}
	pins := make([]int, 0, 6)
	labels := make(map[int]string, 6)
	for i, pin := range cfg.Hardware.Sensors.Approaches() {
		pins = append(pins, pin)
		labels[pin] = names[i] + " Haltelinie"
	}
	pins = append(pins, cfg.Hardware.PowerSwitch, cfg.Hardware.FaultSwitch)
	labels[cfg.Hardware.PowerSwitch] = "Hauptschalter"
	labels[cfg.Hardware.FaultSwitch] = "Notschalter"
	return pins, labels
}

func printEvents(ctx context.Context, out io.Writer, events <-chan treiber.InputEvent, labels map[int]string) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-events:
			state := "offen"
			if event.Active {
				state = "geschlossen"
			}
			fmt.Fprintf(out, "%s  BCM %2d  %-18s %s\n",
				event.Time.Format("15:04:05.000"), event.Pin, labels[event.Pin], state)
		}
	}
}

var approachNames = [4]string{"Nord", "Ost", "Sued", "West"}

var lampNames = [3]string{"Rot", "Gelb", "Gruen"}

type step struct {
	aspects [signal.DirectionCount]signal.Aspect
	hold    time.Duration
}

func showSequence(ctx context.Context, out io.Writer, output *steuerung.Output, timing konfiguration.Timing) error {
	for _, s := range sequenceSteps(timing) {
		if err := output.Show(s.aspects); err != nil {
			return fmt.Errorf("signalbild schalten: %w", err)
		}
		fmt.Fprintf(out, "  %5s  %s\n", s.hold, describe(s.aspects))
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(s.hold):
		}
	}
	return nil
}

func sequenceSteps(timing konfiguration.Timing) []step {
	const (
		ns = 0
		ew = 1
	)
	phase := func(free int, aspect signal.Aspect) [signal.DirectionCount]signal.Aspect {
		aspects := [signal.DirectionCount]signal.Aspect{
			signal.AspectRed, signal.AspectRed, signal.AspectRed, signal.AspectRed,
		}
		if free == ns {
			aspects[signal.North], aspects[signal.South] = aspect, aspect
		} else {
			aspects[signal.East], aspects[signal.West] = aspect, aspect
		}
		return aspects
	}
	allRed := phase(ns, signal.AspectRed)

	steps := []step{{aspects: allRed, hold: timing.AllRed.Duration()}}
	for _, free := range []int{ns, ew} {
		steps = append(steps,
			step{phase(free, signal.AspectRedYellow), timing.RedYellow.Duration()},
			step{phase(free, signal.AspectGreen), timing.BaseGreen.Duration()},
			step{phase(free, signal.AspectYellow), timing.Yellow.Duration()},
			step{allRed, timing.AllRed.Duration()},
		)
	}
	return steps
}

func describe(aspects [signal.DirectionCount]signal.Aspect) string {
	parts := make([]string, 0, signal.DirectionCount)
	for i, aspect := range aspects {
		parts = append(parts, fmt.Sprintf("%s %s", signal.Direction(i), aspect))
	}
	return strings.Join(parts, ", ")
}
