package controller

import (
	"time"

	"ampel/internal/light"
)

// Quittung des Reset-Tasters: dreimal kurz alle Gelblichter. Kurz genug, dass die Vorfuehrung
// nicht stockt, lang genug, um sichtbar zu sein.
const (
	ackFlashes = 3
	ackHalf    = 250 * time.Millisecond
)

// acknowledge ist die Blinkquittung nach einem Reset.
type acknowledge struct {
	running bool
	begun   time.Time
	shown   light.Aspect
	wrote   bool
}

func (a *acknowledge) begin(now time.Time) {
	*a = acknowledge{running: true, begun: now}
}

// next liefert das Signalbild dieses Augenblicks. change sagt, ob es sich vom zuletzt
// gezeigten unterscheidet, done meldet das Ende der Quittung.
func (a *acknowledge) next(now time.Time) (aspect light.Aspect, change, done bool) {
	index := int(now.Sub(a.begun) / ackHalf)
	if index >= 2*ackFlashes {
		a.running = false
		return light.AspectRed, false, true
	}
	aspect = light.AspectOff
	if index%2 == 0 {
		aspect = light.AspectYellowFlash
	}
	if a.wrote && aspect == a.shown {
		return aspect, false, false
	}
	a.shown, a.wrote = aspect, true
	return aspect, true, false
}
