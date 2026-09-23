package config

import (
	"time"

	"ampel/internal/controller"
	"ampel/internal/hal"
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

// Following ist die verkehrsabhaengige Verlaengerung aus der Konfiguration.
func (c *Config) Following() (*strategy.Following, error) {
	return strategy.NewFollowing(
		c.Timing.BaseGreen.Duration(),
		c.Timing.Extension.Duration(),
		c.Timing.MaxGreen.Duration(),
	)
}

// TFTRotation uebersetzt die Angabe aus der Konfiguration in das Register des Controllers.
func (d Display) TFTRotation() int {
	if d.Rotation == "hoch" {
		return hal.RotationPortrait
	}
	return hal.RotationLandscape
}

// Setup uebersetzt die Konfiguration in die Beschreibung der Kreuzung. Hardware, Uhr,
// Strategie und Logging traegt der Aufrufer nach.
func (c *Config) Setup() (controller.Setup, error) {
	return controller.Setup{
		Sensors: c.Hardware.Sensors.Approaches(),
		Timing:  c.ControllerTiming(),
		Follow:  c.Timing.Follow.Duration(),
		Tick:    controller.DefaultTick,
		Sample:  time.Second,
	}, nil
}
