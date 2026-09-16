package controller

import "ampel/internal/detector"

// Feed nimmt eine Flanke auf. Pins, die zu keinem Sensor gehoeren, werden hier ignoriert.
func (c *Controller) Feed(input Input) {
	if c.panel != nil && c.panel.knows(input.Pin) {
		c.panel.level(input.Pin, input.Active)
		return
	}
	if !c.detect.Knows(input.Pin) {
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
		if event.Index == 0 {
			c.lastStopLine[event.Direction] = event.At
		}
		departure, ok := c.approaches[event.Direction].Apply(event, phase)
		if !ok {
			continue
		}
		c.metrics.Add(departure)
		c.observer.VehicleLeft(departure, c.strategy.Name(), Phase(departure.Arrival.Phase).String())
	}
}
