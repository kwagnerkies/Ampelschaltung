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
	"ampel/internal/light"
)

// lampDwell ist die Leuchtdauer je Lampe, lang genug um sie mit dem Auge zu pruefen.
const lampDwell = 400 * time.Millisecond

// selftestBuffer fasst die Flanken, die zwischen zwei Ausgaben anfallen.
const selftestBuffer = 256

// runSelftest prueft die Verdrahtung: erst leuchtet jede Lampe einzeln, danach werden
// Sensorflanken bis zum Abbruch ausgegeben.
func runSelftest(ctx context.Context, cfg *config.Config, out io.Writer) error {
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

	pins, labels := inputPins(cfg)
	inputs, err := hal.NewGPIOInput(cfg.Hardware.Chip, pins, cfg.Hardware.Debounce.Duration(), selftestBuffer, clock.NewReal())
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
	output, err := newOutput(driver, cfg)
	if err != nil {
		return err
	}
	if err := showSequence(ctx, out, output, cfg.Timing); err != nil {
		return err
	}

	fmt.Fprintln(out, "Sensortest, Fahrzeug ueber die Kontakte schieben. Abbruch mit Strg-C.")
	<-ctx.Done()
	fmt.Fprintf(out, "%d Flanken verworfen\n", inputs.Dropped())
	return nil
}

func newOutput(writer controller.LampWriter, cfg *config.Config) (*controller.Output, error) {
	matrix, err := cfg.Hardware.ShiftRegister.LampMatrix()
	if err != nil {
		return nil, err
	}
	bus, err := light.NewBus(matrix, len(cfg.Hardware.ShiftRegister.BitOrder))
	if err != nil {
		return nil, err
	}
	return controller.NewOutput(bus, writer), nil
}

func openLamps(chip *hal.Chip, cfg *config.Config) (hal.LampDriver, error) {
	register := cfg.Hardware.ShiftRegister
	names := [3]string{"data", "clock", "latch"}
	var lines []hal.OutputLine
	for i, pin := range [3]int{register.Data, register.Clock, register.Latch} {
		line, err := chip.Output(pin)
		if err != nil {
			for _, opened := range lines {
				_ = opened.Close()
			}
			return nil, fmt.Errorf("leitung %s: %w", names[i], err)
		}
		lines = append(lines, line)
	}
	return hal.NewShiftRegister(lines[0], lines[1], lines[2], len(register.BitOrder)), nil
}

func walkLamps(ctx context.Context, out io.Writer, driver hal.LampDriver, cfg *config.Config, dwell time.Duration) error {
	positions := cfg.Hardware.ShiftRegister.LampPositions()
	for _, lamp := range config.LampNames() {
		pattern := make([]bool, len(cfg.Hardware.ShiftRegister.BitOrder))
		pattern[positions[lamp]] = true
		if err := driver.Write(pattern); err != nil {
			return fmt.Errorf("lampe %s schalten: %w", lamp, err)
		}
		fmt.Fprintf(out, "  Position %2d  %s\n", positions[lamp], lamp)
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
	pins := make([]int, 0, 14)
	labels := make(map[int]string, 14)
	for i, approach := range cfg.Hardware.Sensors.Approaches() {
		for j, pin := range approach {
			pins = append(pins, pin)
			labels[pin] = fmt.Sprintf("%s Sensor %d", names[i], j)
		}
	}
	pins = append(pins, cfg.Hardware.ModeSwitch, cfg.Hardware.ResetButton)
	labels[cfg.Hardware.ModeSwitch] = "Modus-Kippschalter"
	labels[cfg.Hardware.ResetButton] = "Reset-Taster"
	return pins, labels
}

func printEvents(ctx context.Context, out io.Writer, events <-chan hal.InputEvent, labels map[int]string) {
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
