package hal

import (
	"fmt"

	"github.com/warthog618/go-gpiocdev"
)

// consumer erscheint in gpioinfo und macht belegte Leitungen auf dem Pi zuordenbar.
const consumer = "ampel"

// Chip ist der geoeffnete Zugang zum GPIO-Character-Device.
type Chip struct {
	chip *gpiocdev.Chip
}

func OpenChip(name string) (*Chip, error) {
	chip, err := gpiocdev.NewChip(name, gpiocdev.WithConsumer(consumer))
	if err != nil {
		return nil, fmt.Errorf("gpio-chip %s oeffnen: %w", name, err)
	}
	return &Chip{chip: chip}, nil
}

// Output fordert eine Ausgangsleitung an, die inaktiv startet.
func (c *Chip) Output(pin int) (OutputLine, error) {
	line, err := c.chip.RequestLine(pin, gpiocdev.AsOutput(0))
	if err != nil {
		return nil, fmt.Errorf("ausgang BCM %d anfordern: %w", pin, err)
	}
	return gpioOutput{line: line}, nil
}

func (c *Chip) Name() string { return c.chip.Name }

// Close gibt nur den Zugang zum Chip frei. Angeforderte Leitungen bleiben gueltig und
// werden von ihrem jeweiligen Besitzer geschlossen.
func (c *Chip) Close() error {
	if err := c.chip.Close(); err != nil {
		return fmt.Errorf("gpio-chip schliessen: %w", err)
	}
	return nil
}

type gpioOutput struct {
	line *gpiocdev.Line
}

func (g gpioOutput) Set(high bool) error {
	value := 0
	if high {
		value = 1
	}
	if err := g.line.SetValue(value); err != nil {
		return fmt.Errorf("BCM %d schreiben: %w", g.line.Offset(), err)
	}
	return nil
}

func (g gpioOutput) Close() error {
	if err := g.line.Close(); err != nil {
		return fmt.Errorf("BCM %d freigeben: %w", g.line.Offset(), err)
	}
	return nil
}
