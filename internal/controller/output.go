package controller

import (
	"fmt"

	"ampel/internal/light"
)

// LampCount ist die Zahl der Lampen: drei je Ampelkopf, in der Reihenfolge Rot, Gelb, Gruen.
const LampCount = light.DirectionCount * 3

// LampWriter nimmt den Zustand aller Lampen. Index 0 ist Nord Rot, Index 1 Nord Gelb, und so
// weiter in der Reihenfolge der Zufahrten. Die Schnittstelle steht hier und nicht in der
// Hardwareschicht, damit der Regelkreis nicht von ihr abhaengt.
type LampWriter interface {
	Write(lamps []bool) error
}

// Output ist der einzige Weg zur Lampenhardware. Jedes Muster passiert zuerst die
// Sicherheitspruefung, danach die deutsche Signalfolge der Koepfe.
type Output struct {
	heads  light.Heads
	writer LampWriter
}

func NewOutput(writer LampWriter) *Output {
	return &Output{heads: light.NewHeads(), writer: writer}
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
	if err := o.writer.Write(pattern(aspects)); err != nil {
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

// pattern uebersetzt die Signalbilder in den Zustand der zwoelf Lampen.
func pattern(aspects [light.DirectionCount]light.Aspect) []bool {
	lamps := make([]bool, LampCount)
	for i, aspect := range aspects {
		on := aspect.Lamps()
		lamps[i*3] = on.Red
		lamps[i*3+1] = on.Yellow
		lamps[i*3+2] = on.Green
	}
	return lamps
}
