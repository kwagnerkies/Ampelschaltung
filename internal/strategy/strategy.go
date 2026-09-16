// Paket strategy entscheidet ueber die Dauer der Freigabephasen.
package strategy

import "time"

// View ist der Blick der Strategie auf die Kreuzung. Sie kennt keine Richtungen und keine
// Phasen, nur die freigegebene Seite und die wartende: das haelt die Strategien einfach und
// macht sie ohne Regelkreis testbar.
type View struct {
	Now        time.Time
	GreenSince time.Time
	Target     time.Duration

	OwnDemand   float64
	OtherDemand float64
	OwnQueue    int
	OtherQueue  int

	// LastStopLine ist die letzte Bewegung an einer Haltelinie der freigegebenen Seite. Sie
	// treibt Verlaengerung und Lueckenabbruch.
	LastStopLine time.Duration
	// OtherOldestWait ist die Wartezeit des am laengsten wartenden Fahrzeugs der anderen Seite.
	OtherOldestWait time.Duration
}

// Green ist die bisher verstrichene Freigabezeit.
func (v View) Green() time.Duration { return v.Now.Sub(v.GreenSince) }

// Strategy entscheidet, wie lange eine Freigabe dauert. Die Zwischenzeiten sind fest und
// werden von keiner Strategie veraendert.
type Strategy interface {
	// Name erscheint im Log und in der Auswertung.
	Name() string
	// TargetGreen liefert die Zielgruenzeit beim Wechsel in eine Freigabe.
	TargetGreen(view View) time.Duration
	// EndGreen sagt, ob die laufende Freigabe jetzt endet.
	EndGreen(view View) bool
}
