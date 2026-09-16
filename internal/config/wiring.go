package config

import (
	"fmt"

	"ampel/internal/controller"
	"ampel/internal/strategy"
)

// ControllerTiming sind die Zwischenzeiten des Phasenautomaten.
func (c *Config) ControllerTiming() controller.Timing {
	return controller.Timing{
		Yellow:    c.Timing.Yellow.Duration(),
		AllRed:    c.Timing.AllRed.Duration(),
		RedYellow: c.Timing.RedYellow.Duration(),
	}
}

// StrategyParams sind die Grenzwerte der Regelung.
func (c *Config) StrategyParams() strategy.Params {
	return strategy.Params{
		MinGreen:   c.Timing.MinGreen.Duration(),
		MaxGreen:   c.Timing.MaxGreen.Duration(),
		Cycle:      c.Timing.Cycle.Duration(),
		Intergreen: c.Timing.Intergreen(),
		Gap:        c.Timing.Gap.Duration(),
		Extension:  c.Timing.Extension.Duration(),
		MaxWait:    c.Timing.MaxWait.Duration(),
	}
}

// Setup uebersetzt die Konfiguration in die Beschreibung der Kreuzung. Hardware, Uhr,
// Strategie und Logging traegt der Aufrufer nach.
func (c *Config) Setup() (controller.Setup, error) {
	matrix, err := c.Hardware.ShiftRegister.LampMatrix()
	if err != nil {
		return controller.Setup{}, fmt.Errorf("lampenbelegung: %w", err)
	}
	return controller.Setup{
		Sensors:      c.Hardware.Sensors.Approaches(),
		Debounce:     c.Hardware.Debounce.Duration(),
		QueueMapping: c.QueueMapping,
		DemandAlpha:  c.Adaptive.DemandAlpha,
		LampMatrix:   matrix,
		Bits:         len(c.Hardware.ShiftRegister.BitOrder),
		Timing:       c.ControllerTiming(),
		Tick:         controller.DefaultTick,
		Sample:       c.Logging.StateInterval.Duration(),
	}, nil
}
