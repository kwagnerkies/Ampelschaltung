package config

import (
	"errors"
	"fmt"
)

// Validate prueft die gesamte Konfiguration und meldet alle Verstoesse gesammelt.
func (c *Config) Validate() error {
	var errs []error
	errs = append(errs, c.Hardware.validate(c.Display)...)
	errs = append(errs, c.Timing.validate()...)
	errs = append(errs, c.Display.validate()...)
	return errors.Join(errs...)
}

func (t Timing) validate() []error {
	var errs []error
	durations := []struct {
		name  string
		value Millis
	}{
		{"timing.yellow_ms", t.Yellow},
		{"timing.red_yellow_ms", t.RedYellow},
		{"timing.all_red_ms", t.AllRed},
		{"timing.base_green_ms", t.BaseGreen},
		{"timing.max_green_ms", t.MaxGreen},
		{"timing.follow_ms", t.Follow},
		{"timing.extension_ms", t.Extension},
	}
	for _, d := range durations {
		if d.value <= 0 {
			errs = append(errs, fmt.Errorf("%s muss groesser als null sein", d.name))
		}
	}
	if t.BaseGreen > t.MaxGreen {
		errs = append(errs, fmt.Errorf("timing.base_green_ms (%s) darf timing.max_green_ms (%s) nicht ueberschreiten", t.BaseGreen, t.MaxGreen))
	}
	if t.Extension > t.MaxGreen {
		errs = append(errs, fmt.Errorf("timing.extension_ms (%s) darf timing.max_green_ms (%s) nicht ueberschreiten", t.Extension, t.MaxGreen))
	}
	return errs
}

func (d Display) validate() []error {
	if !d.Enabled {
		return nil
	}
	var errs []error
	if d.Device == "" {
		errs = append(errs, errors.New("display.spi darf nicht leer sein"))
	}
	if d.SpeedHz <= 0 {
		errs = append(errs, errors.New("display.speed_hz muss groesser als null sein"))
	}
	if d.Rotation != "quer" && d.Rotation != "hoch" {
		errs = append(errs, fmt.Errorf("display.rotation ist %q, erlaubt sind quer und hoch", d.Rotation))
	}
	return errs
}
