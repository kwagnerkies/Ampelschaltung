package display

import (
	"time"

	"ampel/internal/controller"
	"ampel/internal/detector"
	"ampel/internal/light"
)

// Observer verbindet die Anzeige mit dem Regelkreis. Sie haengt am selben Beobachter wie das
// CSV-Logging, der Regelkreis kennt sie deshalb nicht.
//
// Ein Sensorereignis zeichnet sofort neu, damit ein auffahrendes Fahrzeug unmittelbar
// sichtbar wird. Der Abtastwert im Sekundentakt haelt die Anzeige auch ohne Verkehr aktuell.
type Observer struct {
	controller.NopObserver
	screen  *Screen
	source  func(time.Time) controller.Snapshot
	onError func(error)
}

var _ controller.Observer = (*Observer)(nil)

// NewObserver erwartet die Quelle des Zustands. Sie wird aus der Goroutine des Regelkreises
// aufgerufen, also aus demselben Besitzer, dem der Zustand gehoert.
func NewObserver(screen *Screen, source func(time.Time) controller.Snapshot, onError func(error)) *Observer {
	return &Observer{screen: screen, source: source, onError: onError}
}

// Source traegt die Quelle nach. Der Regelkreis entsteht erst nach seinem Beobachter.
func (o *Observer) Source(source func(time.Time) controller.Snapshot) { o.source = source }

func (o *Observer) SensorChanged(event detector.SensorEvent) { o.refresh(event.At) }

func (o *Observer) PhaseChanged(at time.Time, _ controller.State, _ string) { o.refresh(at) }

// PowerChanged loescht die Anzeige, wenn die Anlage ausgeschaltet wird.
func (o *Observer) PowerChanged(at time.Time, on bool) {
	if on {
		o.refresh(at)
		return
	}
	if err := o.screen.Clear(); err != nil && o.onError != nil {
		o.onError(err)
	}
}

func (o *Observer) Sample(_ time.Time, snapshot controller.Snapshot) { o.render(snapshot) }

func (o *Observer) refresh(at time.Time) {
	if o.source == nil {
		return
	}
	o.render(o.source(at))
}

// render uebersetzt den Zustand in vier Zahlen. Ein Fehler der Anzeige darf die Kreuzung
// nicht anhalten, deshalb wird er nur gemeldet.
func (o *Observer) render(snapshot controller.Snapshot) {
	var fields [light.DirectionCount]Field
	for _, direction := range light.Directions() {
		fields[direction] = Field{
			Seconds: int(snapshot.Green[direction].Round(time.Second) / time.Second),
			Color:   AspectColor(snapshot.Aspects[direction]),
		}
	}
	if err := o.screen.Update(fields); err != nil && o.onError != nil {
		o.onError(err)
	}
}
