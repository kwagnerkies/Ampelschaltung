// Paket strategy entscheidet ueber die Dauer der Freigabephasen.
package strategy

import "time"

// View ist der Blick der Strategie auf die Kreuzung. Sie kennt keine Richtungen und keine
// Phasen, nur die laufende Freigabe: das haelt die Strategien einfach und ohne Regelkreis
// testbar.
type View struct {
	Now        time.Time
	GreenSince time.Time
	// Following ist die Zahl der Fahrzeuge, die in dieser Freigabe dicht auf ihren Vorgaenger
	// ueber die Haltelinie gefolgt sind. Jedes davon verlaengert die Freigabe.
	Following int
}

// Green ist die bisher verstrichene Freigabezeit.
func (v View) Green() time.Duration { return v.Now.Sub(v.GreenSince) }

// Strategy entscheidet, wie lange eine Freigabe dauert. Die Zwischenzeiten sind fest und
// werden von keiner Strategie veraendert.
type Strategy interface {
	// Name erscheint im Log und in der Auswertung.
	Name() string
	// TargetGreen ist die Gruenzeit, die diese Freigabe nach dem aktuellen Stand bekommt.
	TargetGreen(view View) time.Duration
	// EndGreen sagt, ob die laufende Freigabe jetzt endet.
	EndGreen(view View) bool
}
