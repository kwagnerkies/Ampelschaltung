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
	errs = append(errs, c.Adaptive.validate()...)
	errs = append(errs, c.Logging.validate()...)
	errs = append(errs, c.Learning.validate()...)
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
		{"timing.min_green_ms", t.MinGreen},
		{"timing.max_green_ms", t.MaxGreen},
		{"timing.cycle_ms", t.Cycle},
		{"timing.gap_ms", t.Gap},
		{"timing.extension_ms", t.Extension},
		{"timing.max_wait_ms", t.MaxWait},
	}
	for _, d := range durations {
		if d.value <= 0 {
			errs = append(errs, fmt.Errorf("%s muss groesser als null sein", d.name))
		}
	}
	if t.MinGreen >= t.MaxGreen {
		errs = append(errs, fmt.Errorf("timing.min_green_ms (%s) muss kleiner als timing.max_green_ms (%s) sein", t.MinGreen, t.MaxGreen))
	}
	if effective := t.CycleEffective(); effective < 2*t.MinGreen.Duration() {
		errs = append(errs, fmt.Errorf("timing.cycle_ms laesst nach Abzug der Zwischenzeiten nur %s Gruenzeit, benoetigt werden zweimal timing.min_green_ms (%s)", effective, t.MinGreen))
	}
	if t.Extension > t.MaxGreen {
		errs = append(errs, fmt.Errorf("timing.extension_ms (%s) darf timing.max_green_ms (%s) nicht ueberschreiten", t.Extension, t.MaxGreen))
	}
	// Unterhalb dieser Grenze greift der Verhungerungsschutz noch vor dem ersten regulaeren
	// Wechsel und die Steuerung wechselt dauerhaft mit Mindestgruenzeit.
	if lower := t.MinGreen.Duration() + t.Intergreen(); t.MaxWait.Duration() <= lower {
		errs = append(errs, fmt.Errorf("timing.max_wait_ms (%s) muss groesser als timing.min_green_ms plus Zwischenzeiten (%s) sein", t.MaxWait, lower))
	}
	return errs
}

func (f Fixed) validate(t Timing) []error {
	if f.Green <= 0 {
		return []error{errors.New("fixed.green_ms muss groesser als null sein")}
	}
	if f.Green < t.MinGreen {
		return []error{fmt.Errorf("fixed.green_ms (%s) unterschreitet timing.min_green_ms (%s)", f.Green, t.MinGreen)}
	}
	return nil
}

func (a Adaptive) validate() []error {
	var errs []error
	for _, alpha := range []struct {
		name  string
		value float64
	}{
		{"adaptive.demand_alpha", a.DemandAlpha},
		{"adaptive.learn_alpha", a.LearnAlpha},
	} {
		if alpha.value <= 0 || alpha.value > 1 {
			errs = append(errs, fmt.Errorf("%s muss zwischen null (ausschliesslich) und eins liegen, ist %v", alpha.name, alpha.value))
		}
	}
	if a.BlendK <= 0 {
		errs = append(errs, fmt.Errorf("adaptive.blend_k muss groesser als null sein, ist %v", a.BlendK))
	}
	return errs
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

func (l Learning) validate() []error {
	var errs []error
	if l.Path == "" {
		errs = append(errs, errors.New("learning.path darf nicht leer sein"))
	}
	if l.SaveInterval <= 0 {
		errs = append(errs, errors.New("learning.save_interval_ms muss groesser als null sein"))
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
