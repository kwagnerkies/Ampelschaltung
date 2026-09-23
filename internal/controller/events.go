package controller

import "ampel/internal/detector"

// Feed nimmt eine Flanke auf. Pins, die zu keiner Zufahrt gehoeren, werden hier ignoriert.
func (c *Controller) Feed(input Input) {
	if c.switches != nil && c.switches.knows(input.Pin) {
		c.switches.level(input.Pin, input.Active)
		return
	}
	// Ist die Anlage aus, bewegt sich nichts auf der Kreuzung, was zu erfassen waere.
	if !c.On() || !c.detect.Knows(input.Pin) {
		return
	}
	events, err := c.detect.Feed(input.Pin, input.Active, input.Time)
	if err != nil {
		return
	}
	c.applyEvents(events)
}

// applyEvents wertet die Flanken aus. Gibt ein Fahrzeug die Haltelinie wieder frei, hat es
// die Kreuzung ueberfahren: das ist das Ereignis, das die Freigabe verlaengern kann.
func (c *Controller) applyEvents(events []detector.SensorEvent) {
	for _, event := range events {
		c.observer.SensorChanged(event)
		if !event.Occupied {
			c.countCrossing(event.At, event.Direction)
		}
	}
}
