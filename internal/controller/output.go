package controller

import (
	"ampel/internal/light"
	"errors"
	"fmt"
	"time"
)

const LampCount = light.DirectionCount * 3

type LampWriter interface {
	Write(lamps []bool) error
}

type Output struct {
	heads  light.Heads
	writer LampWriter
}

func NewOutput(writer LampWriter) *Output {
	return &Output{heads: light.NewHeads(), writer: writer}
}

func (o *Output) Aspects() [light.DirectionCount]light.Aspect { return o.heads.Aspects() }

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

func (o *Output) Dark() error {
	var off [light.DirectionCount]light.Aspect
	for i := range off {
		off[i] = light.AspectOff
	}
	return o.Show(off)
}

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

var conflicts = [light.DirectionCount][light.DirectionCount]bool{
	light.North: {light.East: true, light.West: true},
	light.East:  {light.North: true, light.South: true},
	light.South: {light.East: true, light.West: true},
	light.West:  {light.North: true, light.South: true},
}

var ErrConflict = errors.New("unzulaessiger signalzustand")

func Check(aspects [light.DirectionCount]light.Aspect) error {
	for i, own := range aspects {
		if !own.Releasing() {
			continue
		}
		for j, other := range aspects {
			if i == j {
				continue
			}
			if conflicts[i][j] && other.Releasing() {
				return fmt.Errorf("%w: %s zeigt %s, %s zeigt %s",
					ErrConflict, light.Direction(i), own, light.Direction(j), other)
			}
			if other == light.AspectOff {
				return fmt.Errorf("%w: %s zeigt %s, %s ist dunkel",
					ErrConflict, light.Direction(i), own, light.Direction(j))
			}
		}
	}
	return nil
}

func (c *Controller) Shutdown() error {
	state := c.machine.State()
	if state.Stage == StageGreen {
		yellow := State{Phase: state.Phase, Stage: StageYellow}
		if err := c.output.Show(yellow.Aspects()); err != nil {
			return err
		}
	}
	allRed := State{Phase: state.Phase, Stage: StageAllRed}
	if err := c.output.Show(allRed.Aspects()); err != nil {
		return fmt.Errorf("kreuzung auf rot schalten: %w", err)
	}
	return nil
}

func (c *Controller) show() error {
	return c.output.Show(c.machine.State().Aspects())
}

func (c *Controller) emitSample(now time.Time) {
	if c.sample <= 0 || now.Sub(c.lastSample) < c.sample {
		return
	}
	c.lastSample = now
	c.observer.Sample(now, c.Snapshot(now))
}

func (c *Controller) enterFault(now time.Time, err error) {
	c.fault = err
	c.machine.Fault(now)
	c.observer.Fault(now, err)
	c.flashOn = false
	c.flash(now)
}

var errWarning = errors.New("notzustand ueber den schalter")

func (c *Controller) flash(now time.Time) {
	on := (now.Sub(c.machine.State().Since)/c.flashHalf)%2 == 0
	if on == c.flashOn {
		return
	}
	c.flashOn = on
	aspect := light.AspectOff
	if on {
		aspect = light.AspectYellowFlash
	}
	_ = c.showAll(aspect)
}

func (c *Controller) showAll(aspect light.Aspect) error {
	var aspects [light.DirectionCount]light.Aspect
	for direction := range aspects {
		aspects[direction] = aspect
	}
	return c.output.Show(aspects)
}
