package controller

import (
	"time"

	"ampel/internal/light"
	"ampel/internal/mode"
)

// DefaultSwitchDebounce ist die Ruhezeit der Schalter. Ein mechanischer Kippschalter prellt
// laenger als ein Reed-Kontakt, deshalb 100 ms statt der 15 ms der Fahrbahnsensoren.
const DefaultSwitchDebounce = 100 * time.Millisecond

// Switches sind die beiden Kippschalter der Anlage: der Hauptschalter und der Notschalter.
// Geschlossener Kontakt bedeutet jeweils eingeschaltet.
type Switches struct {
	PowerPin int
	FaultPin int
	// PowerOn und FaultOn sind die beim Start gelesenen Pegel.
	PowerOn  bool
	FaultOn  bool
	Debounce time.Duration
}

type switches struct {
	config Switches
	power  *mode.Switch
	fault  *mode.Switch
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
		power:  mode.NewSwitch(config.PowerOn, config.Debounce, now),
		fault:  mode.NewSwitch(config.FaultOn, config.Debounce, now),
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

// On sagt, ob die Anlage eingeschaltet ist. Ohne Schalter laeuft sie immer.
func (c *Controller) On() bool { return c.switches == nil || c.switches.on }

// switchStep fragt beide Schalter ab und meldet, ob der Regelkreis diesen Takt aussetzt.
// Ausgeschaltet ist die Kreuzung dunkel, im Notzustand blinkt sie gelb; in beiden Faellen
// steht der Phasenautomat still.
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
			// Der Zustand ist beendet, also beginnt die Anlage von vorn: Allrot, dann die
			// normale Folge.
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

// restart beginnt bei Allrot. Aus dem dunklen Zustand und aus dem Gelbblinken darf nie
// unmittelbar eine Freigabe folgen.
func (c *Controller) restart(now time.Time) {
	c.fault = nil
	c.flashOn = false
	c.machine.Restart(now)
	c.Reset(now)
	// Waehrend die Anlage stand, hat der Regelkreis nichts getan. Ohne diesen Takt hielte der
	// Watchdog die Pause fuer eine Stoerung.
	c.watchdog.Kick(now)
	c.observer.PowerChanged(now, true)
	if err := c.show(); err != nil {
		c.enterFault(now, err)
		return
	}
	c.observer.PhaseChanged(now, c.machine.State(), "adaptiv")
}
