package controller

import "ampel/internal/detector"

// Feed nimmt eine Flanke auf. Pins, die zu keinem Sensor gehoeren, werden hier ignoriert.
func (c *Controller) Feed(input Input) {
	if c.power != nil && input.Pin == c.power.pin {
		c.power.level = input.Active
		return
	}
	// Ist die Anlage aus, bewegt sich nichts auf der Kreuzung, was zu messen waere.
	if !c.On() || !c.detect.Knows(input.Pin) {
		return
	}
	events, err := c.detect.Feed(input.Pin, input.Active, input.Time)
	if err != nil {
		return
	}
	c.applyEvents(events)
}

func (c *Controller) applyEvents(events []detector.SensorEvent) {
	phase := int(c.machine.State().Phase)
	for _, event := range events {
		c.observer.SensorChanged(event)
		departure, ok := c.approaches[event.Direction].Apply(event, phase)
		if !ok {
			continue
		}
		c.countCrossing(departure.At, departure.Direction)
		c.metrics.Add(departure)
		c.observer.VehicleLeft(departure, c.strategy.Name(), Phase(departure.Arrival.Phase).String())
	}
}
