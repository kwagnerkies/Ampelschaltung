package controller

import (
	"time"

	"ampel/internal/light"
	"ampel/internal/mode"
)

// DefaultPowerDebounce ist die Ruhezeit des Hauptschalters. Ein mechanischer Kippschalter
// prellt laenger als ein Reed-Kontakt, deshalb 100 ms statt der 15 ms der Fahrbahnsensoren.
const DefaultPowerDebounce = 100 * time.Millisecond

// Power ist der Hauptschalter der Anlage. Geschlossener Kontakt bedeutet eingeschaltet.
type Power struct {
	Pin int
	// On ist der beim Start gelesene Pegel.
	On       bool
	Debounce time.Duration
}

// power ist der laufende Zustand des Hauptschalters.
type power struct {
	pin    int
	toggle *mode.Switch
	level  bool
	on     bool
}

func newPower(config Power, now time.Time) *power {
	if config.Debounce <= 0 {
		config.Debounce = DefaultPowerDebounce
	}
	return &power{
		pin:    config.Pin,
		toggle: mode.NewSwitch(config.On, config.Debounce, now),
		level:  config.On,
		on:     config.On,
	}
}

// On sagt, ob die Anlage eingeschaltet ist. Ohne Hauptschalter laeuft sie immer.
func (c *Controller) On() bool { return c.power == nil || c.power.on }

// powerStep fragt den Hauptschalter ab und meldet, ob die Anlage ausgeschaltet ist. Beim
// Ausschalten gehen alle Lichter aus, beim Einschalten beginnt die Anlage mit Allrot und
// einer frischen Messung: aus dem dunklen Zustand darf nie unmittelbar eine Freigabe folgen.
func (c *Controller) powerStep(now time.Time) bool {
	p := c.power
	if p == nil {
		return false
	}
	if p.toggle.Poll(p.level, now) {
		p.on = p.toggle.Level()
		if p.on {
			c.switchOn(now)
		} else {
			c.switchOff(now)
		}
	}
	return !p.on
}

func (c *Controller) switchOff(now time.Time) {
	c.observer.PowerChanged(now, false)
	if err := c.showAll(light.AspectOff); err != nil {
		c.enterFault(now, err)
	}
}

// switchOn beginnt einen neuen Lauf. Der Hauptschalter ist damit zugleich der Weg, eine
// Messung sauber zu trennen, und er holt die Anlage aus dem Notzustand.
func (c *Controller) switchOn(now time.Time) {
	c.fault = nil
	// Waehrend die Anlage aus war, hat der Regelkreis nichts getan. Ohne diesen Takt haelte
	// der Watchdog die Pause fuer eine Stoerung.
	c.watchdog.Kick(now)
	c.flashOn = false
	c.machine.Restart(now)
	c.Reset(now)
	c.observer.PowerChanged(now, true)
	if err := c.show(); err != nil {
		c.enterFault(now, err)
		return
	}
	c.observer.PhaseChanged(now, c.machine.State(), c.strategy.Name())
}
