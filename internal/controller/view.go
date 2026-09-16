package controller

import (
	"time"

	"ampel/internal/light"
	"ampel/internal/strategy"
)

// view ist der Blick der Strategie auf die laufende Freigabe.
func (c *Controller) view(now time.Time) strategy.View {
	return strategy.View{
		Now:        now,
		GreenSince: c.machine.State().Since,
		Following:  c.following,
	}
}

// countCrossing zaehlt ein Fahrzeug, das die Haltelinie einer freigegebenen Zufahrt
// ueberfahren hat. Verglichen wird je Zufahrt: zwei Fahrzeuge gelten nur dann als Folge, wenn
// sie hintereinander ueber dieselbe Haltelinie fahren. Die gegenueberliegende Zufahrt faehrt
// gleichzeitig ab, ihre Abfahrten sind keine Folge.
func (c *Controller) countCrossing(at time.Time, direction light.Direction) {
	state := c.machine.State()
	if state.Stage != StageGreen || PhaseOf(direction) != state.Phase {
		return
	}
	if last := c.lastCrossing[direction]; !last.IsZero() && at.Sub(last) <= c.follow {
		c.following++
	}
	c.lastCrossing[direction] = at
}

// Snapshot ist der aktuelle Zustand fuer Log und Anzeige.
func (c *Controller) Snapshot(now time.Time) Snapshot {
	state := c.machine.State()
	snapshot := Snapshot{
		State:     state,
		Elapsed:   now.Sub(state.Since),
		Mode:      c.strategy.Name(),
		Following: c.following,
		Aspects:   state.Aspects(),
	}
	for _, direction := range light.Directions() {
		snapshot.Reach[direction] = c.approaches[direction].Reach()
		if PhaseOf(direction) == state.Phase && state.Stage == StageGreen {
			// Die freigegebene Richtung zeigt die Restzeit. Sie zaehlt herunter und springt
			// hoch, sobald ein dicht folgendes Fahrzeug die Freigabe verlaengert.
			if remaining := state.Target - snapshot.Elapsed; remaining > 0 {
				snapshot.Green[direction] = remaining
			}
		} else {
			// Die wartende Richtung zeigt ihre Grundzeit: verlaengert wird erst, wenn dort
			// tatsaechlich Fahrzeuge fahren.
			snapshot.Green[direction] = c.strategy.TargetGreen(strategy.View{Now: now, GreenSince: now})
		}
	}
	return snapshot
}
