package controller

import (
	"time"

	"ampel/internal/light"
)

const DefaultSwitchDebounce = 100 * time.Millisecond

type Switches struct {
	PowerPin int
	FaultPin int
	PowerOn  bool
	FaultOn  bool
	Debounce time.Duration
}

type switches struct {
	config Switches
	power  *toggle
	fault  *toggle
	levels map[int]bool
	on     bool
	warn   bool
}

func newSwitches(config Switches, now time.Time) *switches {
	if config.Debounce <= 0 {
		config.Debounce = DefaultSwitchDebounce
	}
	return &switches{
		config: config,
		power:  newToggle(config.PowerOn, config.Debounce, now),
		fault:  newToggle(config.FaultOn, config.Debounce, now),
		levels: map[int]bool{config.PowerPin: config.PowerOn, config.FaultPin: config.FaultOn},
		on:     config.PowerOn,
		warn:   config.FaultOn,
	}
}

func (s *switches) knows(pin int) bool {
	_, ok := s.levels[pin]
	return ok
}

func (s *switches) level(pin int, active bool) { s.levels[pin] = active }

func (c *Controller) On() bool { return c.switches == nil || c.switches.on }

func (c *Controller) switchStep(now time.Time) bool {
	s := c.switches
	if s == nil {
		return false
	}
	if s.power.Poll(s.levels[s.config.PowerPin], now) {
		s.on = s.power.Level()
		if s.on {
			c.restart(now)
		} else {
			c.switchOff(now)
		}
	}
	if !s.on {
		return true
	}
	if s.fault.Poll(s.levels[s.config.FaultPin], now) {
		s.warn = s.fault.Level()
		if s.warn {
			c.enterFault(now, errWarning)
		} else {
			c.restart(now)
		}
	}
	if s.warn {
		c.flash(now)
		return true
	}
	return false
}

func (c *Controller) switchOff(now time.Time) {
	c.observer.PowerChanged(now, false)
	if err := c.showAll(light.AspectOff); err != nil {
		c.enterFault(now, err)
	}
}

func (c *Controller) restart(now time.Time) {
	c.fault = nil
	c.flashOn = false
	c.machine.Restart(now)
	c.Reset(now)
	c.observer.PowerChanged(now, true)
	if err := c.show(); err != nil {
		c.enterFault(now, err)
		return
	}
	c.observer.PhaseChanged(now, c.machine.State())
}

type toggle struct {
	debounce time.Duration
	level    bool
	raw      bool
	rawAt    time.Time
}

func newToggle(initial bool, debounce time.Duration, now time.Time) *toggle {
	return &toggle{debounce: debounce, level: initial, raw: initial, rawAt: now}
}

func (s *toggle) Poll(level bool, now time.Time) bool {
	if level != s.raw {
		s.raw = level
		s.rawAt = now
		return false
	}
	if s.raw == s.level || now.Sub(s.rawAt) < s.debounce {
		return false
	}
	s.level = s.raw
	return true
}

func (s *toggle) Level() bool { return s.level }
