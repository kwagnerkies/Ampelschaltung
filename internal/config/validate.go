package config

import (
	"errors"
	"fmt"
)

// Validate prueft die gesamte Konfiguration und meldet alle Verstoesse gesammelt.
func (c *Config) Validate() error {
	var errs []error
	errs = append(errs, c.Hardware.validate()...)
	errs = append(errs, c.Timing.validate()...)
	errs = append(errs, c.Fixed.validate(c.Timing)...)
	errs = append(errs, c.Logging.validate()...)
	errs = append(errs, validateQueueMapping(c.QueueMapping, c.Hardware.Sensors.SensorCount())...)
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

func (f Fixed) validate(t Timing) []error {
	if f.Green <= 0 {
		return []error{errors.New("fixed.green_ms muss groesser als null sein")}
	}
	if f.Green > t.MaxGreen {
		return []error{fmt.Errorf("fixed.green_ms (%s) ueberschreitet timing.max_green_ms (%s)", f.Green, t.MaxGreen)}
	}
	return nil
}

func (l Logging) validate() []error {
	var errs []error
	if l.Dir == "" {
		errs = append(errs, errors.New("logging.dir darf nicht leer sein"))
	}
	if l.StateInterval <= 0 {
		errs = append(errs, errors.New("logging.state_interval_ms muss groesser als null sein"))
	}
	if l.Buffer <= 0 {
		errs = append(errs, errors.New("logging.buffer muss groesser als null sein"))
	}
	return errs
}

func validateQueueMapping(mapping map[int]int, sensorCount int) []error {
	var errs []error
	if sensorCount <= 0 {
		return nil
	}
	previous := -1
	for occupied := 0; occupied <= sensorCount; occupied++ {
		vehicles, ok := mapping[occupied]
		if !ok {
			errs = append(errs, fmt.Errorf("queue_mapping fehlt der Eintrag fuer %d belegte Sensoren", occupied))
			continue
		}
		if vehicles < 0 {
			errs = append(errs, fmt.Errorf("queue_mapping[%d] darf nicht negativ sein", occupied))
		}
		if vehicles < previous {
			errs = append(errs, fmt.Errorf("queue_mapping[%d] (%d) ist kleiner als der Eintrag davor (%d)", occupied, vehicles, previous))
		}
		previous = vehicles
	}
	if mapping[0] != 0 {
		errs = append(errs, errors.New("queue_mapping[0] muss null Fahrzeuge ergeben"))
	}
	for occupied := range mapping {
		if occupied < 0 || occupied > sensorCount {
			errs = append(errs, fmt.Errorf("queue_mapping[%d] liegt ausserhalb der Sensoranzahl %d", occupied, sensorCount))
		}
	}
	return errs
}
