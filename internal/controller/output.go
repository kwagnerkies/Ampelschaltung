package controller

import (
	"fmt"

	"ampel/internal/light"
)

// LampWriter nimmt das Bitmuster der Lampenkette. Die Schnittstelle steht hier und nicht in
// der Hardwareschicht, damit der Regelkreis nicht von ihr abhaengt.
type LampWriter interface {
	Write(pattern []bool) error
}

// Output ist der einzige Weg zur Lampenhardware. Jedes Muster passiert zuerst die
// Sicherheitspruefung, danach die deutsche Signalfolge der Koepfe.
type Output struct {
	heads  light.Heads
	bus    *light.Bus
	writer LampWriter
}

func NewOutput(bus *light.Bus, writer LampWriter) *Output {
	return &Output{heads: light.NewHeads(), bus: bus, writer: writer}
}

func (o *Output) Aspects() [light.DirectionCount]light.Aspect { return o.heads.Aspects() }

// Show uebernimmt neue Signalbilder. Bei einem Konflikt oder einem Verstoss gegen die
// Signalfolge bleibt der bisherige Zustand stehen und die Hardware wird nicht beschrieben.
func (o *Output) Show(aspects [light.DirectionCount]light.Aspect) error {
	if err := Check(aspects); err != nil {
		return err
	}
	if err := o.heads.Set(aspects); err != nil {
		return err
	}
	if err := o.writer.Write(o.bus.Pattern(aspects)); err != nil {
		return fmt.Errorf("lampen schreiben: %w", err)
	}
	return nil
}

// Dark schaltet alle Koepfe ab. Fuer das geordnete Beenden, wenn kein Signalbild mehr gilt.
func (o *Output) Dark() error {
	var off [light.DirectionCount]light.Aspect
	for i := range off {
		off[i] = light.AspectOff
	}
	return o.Show(off)
}
