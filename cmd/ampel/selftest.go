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
	if err := showSequence(ctx, out, controller.NewOutput(driver), cfg.Timing); err != nil {
		return err
	}

	fmt.Fprintln(out, "Sensortest, Fahrzeug ueber die Kontakte schieben. Abbruch mit Strg-C.")
	<-ctx.Done()
	fmt.Fprintf(out, "%d Flanken verworfen\n", inputs.Dropped())
	return nil
}

// openLamps fordert die zwoelf LED-Leitungen an, in der Reihenfolge Nord Rot, Nord Gelb,
// Nord Gruen, dann Ost und so weiter.
func openLamps(chip *hal.Chip, cfg *config.Config) (hal.LampDriver, error) {
	var lines []hal.OutputLine
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
	return hal.NewLamps(lines), nil
}

func walkLamps(ctx context.Context, out io.Writer, driver hal.LampDriver, cfg *config.Config, dwell time.Duration) error {
	for index := 0; index < controller.LampCount; index++ {
		pattern := make([]bool, controller.LampCount)
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

var approachNames = [4]string{"Nord", "Ost", "Sued", "West"}

var lampNames = [3]string{"Rot", "Gelb", "Gruen"}
