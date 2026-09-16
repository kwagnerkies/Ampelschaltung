// Paket light beschreibt Signalbilder, Ampelkoepfe und ihre Abbildung auf Registerbits.
package light

import "slices"

// Aspect ist das Signalbild eines Ampelkopfes. Rot ist der Nullwert, damit ein nicht
// gesetzter Kopf nicht versehentlich freigibt.
type Aspect uint8

const (
	AspectRed Aspect = iota
	AspectRedYellow
	AspectGreen
	AspectYellow
	AspectOff
	// AspectYellowFlash ist die Hellphase des Notzustands. Das Blinken selbst entsteht durch
	// den Wechsel mit AspectOff im Takt von 1 Hz.
	AspectYellowFlash
)

// Lamps sind die drei Lampen eines Kopfes. Rot und Gelb leuchten in der deutschen
// Signalfolge zeitweise gemeinsam.
type Lamps struct {
	Red    bool
	Yellow bool
	Green  bool
}

func (a Aspect) Lamps() Lamps {
	switch a {
	case AspectRed:
		return Lamps{Red: true}
	case AspectRedYellow:
		return Lamps{Red: true, Yellow: true}
	case AspectGreen:
		return Lamps{Green: true}
	case AspectYellow, AspectYellowFlash:
		return Lamps{Yellow: true}
	}
	return Lamps{}
}

// successors ist die deutsche Signalfolge: Rot, Rot und Gelb, Gruen, Gelb, Rot. Aus dem
// abgeschalteten Zustand wird zuerst Rot gezeigt, nie sofort Gruen.
var successors = map[Aspect][]Aspect{
	AspectRed: {AspectRedYellow},
	// Aus Rot und Gelb zurueck auf Rot ist zulaessig, weil noch nichts freigegeben war. Das
	// braucht das geordnete Herunterfahren mitten in einem Wechsel.
	AspectRedYellow:   {AspectGreen, AspectRed},
	AspectGreen:       {AspectYellow},
	AspectYellow:      {AspectRed},
	AspectOff:         {AspectRed},
	AspectYellowFlash: {AspectRed},
}

// CanFollow sagt, ob dieses Signalbild unmittelbar auf previous folgen darf. Halten ist immer
// erlaubt, ebenso der Wechsel in den Notzustand und das Abschalten.
func (a Aspect) CanFollow(previous Aspect) bool {
	if a == previous || a == AspectYellowFlash || a == AspectOff {
		return true
	}
	return slices.Contains(successors[previous], a)
}

// Releasing gilt fuer alle Signalbilder, in denen eine Zufahrt fahren darf oder unmittelbar
// davor steht. Gelb zaehlt dazu, weil dann noch Fahrzeuge in der Kreuzung stehen.
func (a Aspect) Releasing() bool {
	switch a {
	case AspectRedYellow, AspectGreen, AspectYellow:
		return true
	}
	return false
}

func (a Aspect) String() string {
	switch a {
	case AspectRed:
		return "Rot"
	case AspectRedYellow:
		return "RotGelb"
	case AspectGreen:
		return "Gruen"
	case AspectYellow:
		return "Gelb"
	case AspectOff:
		return "Aus"
	case AspectYellowFlash:
		return "GelbBlinken"
	}
	return "unbekannt"
}
