package controller

import (
	"time"

	"ampel/internal/mode"
	"ampel/internal/strategy"
)

// Ruhezeiten der Bedienelemente. Ein mechanischer Kippschalter prellt laenger als ein
// Reed-Kontakt, deshalb 100 ms statt der 15 ms der Fahrbahnsensoren. Der Reset loest erst
// nach zwei Sekunden Dauerdruck aus, damit ein versehentlicher Druck waehrend der
// Vorfuehrung den Lernzustand nicht loescht.
const (
	DefaultSwitchDebounce = 100 * time.Millisecond
	DefaultResetHold      = 2 * time.Second
)

// Panel beschreibt Kippschalter und Reset-Taster. Der geschlossene Kippschalter waehlt die
// adaptive Steuerung, der offene die Festzeitsteuerung.
type Panel struct {
	SwitchPin int
	ResetPin  int
	Fixed     strategy.Strategy
	Adaptive  strategy.Strategy
	// SwitchClosed ist der beim Start gelesene Pegel. Ohne ihn wuerde der erste Wechsel des
	// Schalters erst nach einer Flanke bemerkt.
	SwitchClosed bool
	Debounce     time.Duration
	Hold         time.Duration
}

// panel ist der laufende Zustand der Bedienelemente. Die Pegel kommen aus den Flanken der
// Hardwareschicht, abgefragt wird zyklisch im Takt des Regelkreises.
type panel struct {
	config  Panel
	toggle  *mode.Switch
	button  *mode.Button
	closed  bool
	pressed bool
	pending strategy.Strategy
	ack     acknowledge
	// resetAt ist der Zeitpunkt des ausgeloesten Resets, solange die Quittung noch aussteht.
	resetAt time.Time
}

func newPanel(config Panel, now time.Time) *panel {
	if config.Debounce <= 0 {
		config.Debounce = DefaultSwitchDebounce
	}
	if config.Hold <= 0 {
		config.Hold = DefaultResetHold
	}
	return &panel{
		config: config,
		toggle: mode.NewSwitch(config.SwitchClosed, config.Debounce, now),
		button: mode.NewButton(config.Hold),
		closed: config.SwitchClosed,
	}
}

func (p *panel) knows(pin int) bool {
	return pin == p.config.SwitchPin || pin == p.config.ResetPin
}

func (p *panel) level(pin int, active bool) {
	switch pin {
	case p.config.SwitchPin:
		p.closed = active
	case p.config.ResetPin:
		p.pressed = active
	}
}

// poll fragt beide Elemente ab und meldet den langen Druck auf den Reset-Taster. Ein Wechsel
// des Kippschalters wird nur vorgemerkt.
func (p *panel) poll(now time.Time) bool {
	if p.toggle.Poll(p.closed, now) {
		p.pending = p.config.Fixed
		if p.toggle.Level() {
			p.pending = p.config.Adaptive
		}
	}
	return p.button.Poll(p.pressed, now)
}

// panelStep fragt die Bedienelemente ab und fuehrt die Blinkquittung aus. Solange sie laeuft,
// steht der Phasenautomat still. Sie beginnt nur aus Allrot heraus, damit nie ein Blinken
// eine laufende Freigabe ueberdeckt.
func (c *Controller) panelStep(now time.Time) bool {
	p := c.panel
	if p == nil {
		return false
	}
	if p.ack.running {
		return c.blink(now)
	}
	if p.poll(now) {
		c.Reset(now)
		c.observer.Reset(now)
		p.resetAt = now
	}
	if p.resetAt.IsZero() || c.machine.State().Stage != StageAllRed {
		return false
	}
	p.resetAt = time.Time{}
	p.ack.begin(now)
	return c.blink(now)
}

// blink zeigt den laufenden Impuls der Quittung und meldet, ob sie weiterlaeuft.
func (c *Controller) blink(now time.Time) bool {
	aspect, change, done := c.panel.ack.next(now)
	switch {
	case done:
		// Nach der Quittung gilt die volle Allrotzeit, bevor die naechste Freigabe beginnt.
		c.machine.Hold(now)
		if err := c.show(); err != nil {
			c.enterFault(now, err)
		}
		return false
	case change:
		if err := c.showAll(aspect); err != nil {
			c.enterFault(now, err)
			return false
		}
	}
	return true
}

// resetWaiting beendet eine laufende Freigabe vorzeitig, damit die Quittung bald aus Allrot
// heraus laufen kann. Eine Freigabe, die erst nach dem Tastendruck begann, bleibt unberuehrt:
// sonst entstuende ein Gruen von wenigen Millisekunden.
func (c *Controller) resetWaiting(state State) bool {
	return c.panel != nil && !c.panel.resetAt.IsZero() && state.Since.Before(c.panel.resetAt)
}

// applyMode uebernimmt die am Kippschalter gewaehlte Betriebsart. Der Wechsel wirkt erst beim
// Eintritt in eine Freigabe: waehrend Gelb oder Allrot umzuschalten koennte ein unzulaessiges
// Signalbild erzeugen.
func (c *Controller) applyMode(now time.Time) {
	if c.panel == nil || c.panel.pending == nil {
		return
	}
	next := c.panel.pending
	c.panel.pending = nil
	if next == nil || next == c.strategy {
		return
	}
	c.strategy = next
	c.observer.ModeChanged(now, next.Name())
}

// learns sagt, ob die laufende Betriebsart das Tagesprofil mitlernt. Die Festzeitsteuerung
// misst die Wartezeiten mit, lernt aber nicht.
func (c *Controller) learns() bool {
	return c.panel == nil || c.strategy != c.panel.config.Fixed
}
