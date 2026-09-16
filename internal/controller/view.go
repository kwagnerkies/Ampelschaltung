package controller

import (
	"time"

	"ampel/internal/light"
	"ampel/internal/strategy"
)

// view baut den Blick der Strategie auf die Kreuzung. Nachfrage einer Phase ist das Maximum
// der beteiligten Zufahrten, nicht die Summe: massgeblich ist der schlechteste Arm.
func (c *Controller) view(now time.Time) strategy.View {
	state := c.machine.State()
	own, other := state.Phase, state.Phase.Other()
	return strategy.View{
		Now:             now,
		GreenSince:      state.Since,
		Target:          state.Target,
		OwnDemand:       c.demand(now, own),
		OtherDemand:     c.demand(now, other),
		OwnQueue:        c.queueLength(own),
		OtherQueue:      c.queueLength(other),
		LastStopLine:    c.sinceStopLine(now, own),
		OtherOldestWait: c.oldestWait(now, other),
	}
}

// demand ist die Nachfrage einer Phase: das Maximum der beteiligten Zufahrten, jeweils
// gemischt aus Messung und Prognose.
func (c *Controller) demand(now time.Time, phase Phase) float64 {
	worst := 0.0
	for _, direction := range phase.Directions() {
		blended, _ := c.learner.Blend(now, direction, c.approaches[direction].Demand())
		if blended > worst {
			worst = blended
		}
	}
	return worst
}

// weight ist das Prognosegewicht der freigegebenen Phase. Es zeigt im Log, wie weit das
// Lernen fortgeschritten ist.
func (c *Controller) weight(now time.Time, phase Phase) float64 {
	var highest float64
	for _, direction := range phase.Directions() {
		if _, w := c.learner.Blend(now, direction, 0); w > highest {
			highest = w
		}
	}
	return highest
}

func (c *Controller) queueLength(phase Phase) int {
	worst := 0
	for _, direction := range phase.Directions() {
		if queue := c.approaches[direction].QueueLength(); queue > worst {
			worst = queue
		}
	}
	return worst
}

func (c *Controller) oldestWait(now time.Time, phase Phase) time.Duration {
	var worst time.Duration
	for _, direction := range phase.Directions() {
		if wait := c.approaches[direction].OldestWait(now); wait > worst {
			worst = wait
		}
	}
	return worst
}

// sinceStopLine ist die Zeit seit der letzten Bewegung an einer Haltelinie der freigegebenen
// Zufahrten.
func (c *Controller) sinceStopLine(now time.Time, phase Phase) time.Duration {
	var latest time.Time
	for _, direction := range phase.Directions() {
		if at := c.lastStopLine[direction]; at.After(latest) {
			latest = at
		}
	}
	if latest.IsZero() {
		latest = c.started
	}
	return now.Sub(latest)
}

// Snapshot ist der aktuelle Zustand fuer das Log.
func (c *Controller) Snapshot(now time.Time) Snapshot {
	state := c.machine.State()
	snapshot := Snapshot{
		State:   state,
		Elapsed: now.Sub(state.Since),
		Mode:    c.strategy.Name(),
		Weight:  c.weight(now, state.Phase),
	}
	for _, direction := range light.Directions() {
		snapshot.Queues[direction] = c.approaches[direction].QueueLength()
		snapshot.Demands[direction] = c.approaches[direction].Demand()
	}
	return snapshot
}
