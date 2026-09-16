package strategy

import (
	"fmt"
	"time"
)

// Params sind die Grenzwerte der Regelung. Sie stammen aus der Konfiguration, stehen hier
// aber als eigener Typ, damit die Strategie die Konfiguration nicht kennen muss.
type Params struct {
	MinGreen   time.Duration
	MaxGreen   time.Duration
	Cycle      time.Duration
	Intergreen time.Duration
	Gap        time.Duration
	Extension  time.Duration
	MaxWait    time.Duration
}

func (p Params) Validate() error {
	if p.MinGreen <= 0 || p.MaxGreen <= p.MinGreen {
		return fmt.Errorf("gruenzeiten: min %s, max %s", p.MinGreen, p.MaxGreen)
	}
	if p.Gap <= 0 || p.Extension <= 0 {
		return fmt.Errorf("luecke %s, verlaengerung %s", p.Gap, p.Extension)
	}
	if p.MaxWait <= p.MinGreen {
		return fmt.Errorf("hoechstwartezeit %s liegt nicht ueber der mindestgruenzeit %s", p.MaxWait, p.MinGreen)
	}
	return nil
}

// CycleEffective ist die Umlaufzeit abzueglich der Zwischenzeiten beider Wechsel. Nur diese
// Zeit steht als Gruenzeit zur Verteilung bereit.
func (p Params) CycleEffective() time.Duration {
	effective := p.Cycle - 2*p.Intergreen
	if effective < 0 {
		return 0
	}
	return effective
}

func (p Params) clamp(d time.Duration) time.Duration {
	switch {
	case d < p.MinGreen:
		return p.MinGreen
	case d > p.MaxGreen:
		return p.MaxGreen
	}
	return d
}
