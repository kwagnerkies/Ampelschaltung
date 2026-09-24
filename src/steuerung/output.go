package steuerung

import (
	"ampel/src/signal"
	"errors"
	"fmt"
	"time"
)

const LampCount = signal.DirectionCount * 3

type LampWriter interface {
	Write(lamps []bool) error
}

type Output struct {
	heads  signal.Heads
	writer LampWriter
}

func NewOutput(writer LampWriter) *Output {
	return &Output{heads: signal.NewHeads(), writer: writer}
}

func (o *Output) Aspects() [signal.DirectionCount]signal.Aspect { return o.heads.Aspects() }

func (o *Output) Show(aspects [signal.DirectionCount]signal.Aspect) error {
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
	var off [signal.DirectionCount]signal.Aspect
	for i := range off {
		off[i] = signal.AspectOff
	}
	return o.Show(off)
}

func pattern(aspects [signal.DirectionCount]signal.Aspect) []bool {
	lamps := make([]bool, LampCount)
	for i, aspect := range aspects {
		on := aspect.Lamps()
		lamps[i*3] = on.Red
		lamps[i*3+1] = on.Yellow
		lamps[i*3+2] = on.Green
	}
	return lamps
}

var conflicts = [signal.DirectionCount][signal.DirectionCount]bool{
	signal.North: {signal.East: true, signal.West: true},
	signal.East:  {signal.North: true, signal.South: true},
	signal.South: {signal.East: true, signal.West: true},
	signal.West:  {signal.North: true, signal.South: true},
}

var ErrConflict = errors.New("unzulaessiger signalzustand")

func Check(aspects [signal.DirectionCount]signal.Aspect) error {
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
					ErrConflict, signal.Direction(i), own, signal.Direction(j), other)
			}
			if other == signal.AspectOff {
				return fmt.Errorf("%w: %s zeigt %s, %s ist dunkel",
					ErrConflict, signal.Direction(i), own, signal.Direction(j))
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
	aspect := signal.AspectOff
	if on {
		aspect = signal.AspectYellowFlash
	}
	_ = c.showAll(aspect)
}

func (c *Controller) showAll(aspect signal.Aspect) error {
	var aspects [signal.DirectionCount]signal.Aspect
	for direction := range aspects {
		aspects[direction] = aspect
	}
	return c.output.Show(aspects)
}
