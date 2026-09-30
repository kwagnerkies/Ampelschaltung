package main

import (
	"ampel/src/clock"
	"ampel/src/config"
	"ampel/src/controller"
	"ampel/src/driver"
	"ampel/src/driver/gpio"
	"ampel/src/driver/spi"
	"ampel/src/driver/ws2812"
	"ampel/src/light"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

const lampDwell = 400 * time.Millisecond

const selftestBuffer = 256

func runSelftest(ctx context.Context, cfg *config.Config, out io.Writer) error {
	chip, err := gpio.OpenChip(cfg.Hardware.Chip)
	if err != nil {
		return err
	}
	defer func() { _ = chip.Close() }()

	driver, err := openLamps(cfg)
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
	if err := walkLamps(ctx, out, driver, cfg.Hardware.Lamps.Pixels, lampDwell); err != nil {
		return err
	}

	fmt.Fprintln(out, "Signalfolge, beide Freigabephasen einmal:")
	if err := showSequence(ctx, out, controller.NewOutput(driver), cfg.Timing); err != nil {
		return err
	}

	fmt.Fprintln(out, "Sensortest, Fahrzeug ueber die Kontakte schieben. Abbruch mit Strg-C.")
	<-ctx.Done()
	fmt.Fprintf(out, "%d Flanken verworfen\n", inputs.Dropped())
	return nil
}

func openLamps(cfg *config.Config) (driver.LampDriver, error) {
	bus, err := spi.OpenSPI(cfg.Hardware.Lamps.Device, cfg.Hardware.Lamps.SpeedHz)
	if err != nil {
		return nil, err
	}
	strip, err := ws2812.New(bus, light.DirectionCount, byte(cfg.Hardware.Lamps.Brightness), cfg.Hardware.Lamps.Pixels)
	if err != nil {
		_ = bus.Close()
		return nil, err
	}
	return strip, nil
}

func walkLamps(ctx context.Context, out io.Writer, driver driver.LampDriver, pixels [3]int, dwell time.Duration) error {
	for index := 0; index < controller.LampCount; index++ {
		pattern := make([]bool, controller.LampCount)
		pattern[index] = true
		if err := driver.Write(pattern); err != nil {
			return fmt.Errorf("lampe %d schalten: %w", index, err)
		}
		fmt.Fprintf(out, "  Stick %d Pixel %d  %s %s\n",
			index/3, pixels[index%3], approachNames[index/3], lampNames[index%3])
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

func inputPins(cfg *config.Config) ([]int, map[int]string) {
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

func printEvents(ctx context.Context, out io.Writer, events <-chan driver.InputEvent, labels map[int]string) {
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
	aspects [light.DirectionCount]light.Aspect
	hold    time.Duration
}

func showSequence(ctx context.Context, out io.Writer, output *controller.Output, timing config.Timing) error {
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

func sequenceSteps(timing config.Timing) []step {
	const (
		ns = 0
		ew = 1
	)
	phase := func(free int, aspect light.Aspect) [light.DirectionCount]light.Aspect {
		aspects := [light.DirectionCount]light.Aspect{
			light.AspectRed, light.AspectRed, light.AspectRed, light.AspectRed,
		}
		if free == ns {
			aspects[light.North], aspects[light.South] = aspect, aspect
		} else {
			aspects[light.East], aspects[light.West] = aspect, aspect
		}
		return aspects
	}
	allRed := phase(ns, light.AspectRed)

	steps := []step{{aspects: allRed, hold: timing.AllRed.Duration()}}
	for _, free := range []int{ns, ew} {
		steps = append(steps,
			step{phase(free, light.AspectRedYellow), timing.RedYellow.Duration()},
			step{phase(free, light.AspectGreen), timing.BaseGreen.Duration()},
			step{phase(free, light.AspectYellow), timing.Yellow.Duration()},
			step{allRed, timing.AllRed.Duration()},
		)
	}
	return steps
}

func describe(aspects [light.DirectionCount]light.Aspect) string {
	parts := make([]string, 0, light.DirectionCount)
	for i, aspect := range aspects {
		parts = append(parts, fmt.Sprintf("%s %s", light.Direction(i), aspect))
	}
	return strings.Join(parts, ", ")
}
