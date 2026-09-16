package controller

import (
	"fmt"
	"time"

	"ampel/internal/light"
)

// Shutdown fuehrt die Kreuzung auf Rot. Aus einer Freigabe fuehrt der Weg ueber Gelb, sonst
// entsteht ein unzulaessiges Signalbild. Die Zwischenzeit wird dabei nicht abgewartet, denn
// der Prozess endet ohnehin.
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

// flash laesst alle Lichter mit 1 Hz gelb blinken. Geschrieben wird nur beim Wechsel.
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

// showAll zeigt auf allen Koepfen dasselbe Signalbild. Das brauchen Notzustand und
// Blinkquittung, beide sind keine Phase des Automaten.
func (c *Controller) showAll(aspect light.Aspect) error {
	var aspects [light.DirectionCount]light.Aspect
	for direction := range aspects {
		aspects[direction] = aspect
	}
	return c.output.Show(aspects)
}
