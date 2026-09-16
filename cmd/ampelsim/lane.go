package main

import (
	"time"

	"ampel/internal/controller"
	"ampel/internal/light"
)

// Zeitkonstanten des Fahrzeugmodells. Sie beschreiben ein Modellauto auf der gedruckten
// Platte, nicht ein Fahrzeug im Strassenverkehr.
const (
	// crossingTime ist der Zeitbedarf eines Fahrzeugs ueber die Haltelinie.
	crossingTime = 800 * time.Millisecond
	// shiftTime ist die Zeit, bis das naechste Fahrzeug an die Linie vorgerueckt ist.
	shiftTime = 600 * time.Millisecond
)

// lane ist eine simulierte Zufahrt. Die Warteschlange steht dicht auf: das n-te Fahrzeug
// belegt den n-ten Sensor, weiter hinten stehende Fahrzeuge sieht kein Sensor.
type lane struct {
	direction light.Direction
	pins      []int
	queue     []time.Time
	occupied  []bool
	atLine    bool
	since     time.Time
	lastLeft  time.Time

	arrived  int
	departed int
	waitSum  time.Duration
	waitMax  time.Duration
}

func newLane(direction light.Direction, pins []int, now time.Time) *lane {
	return &lane{
		direction: direction,
		pins:      pins,
		occupied:  make([]bool, len(pins)),
		lastLeft:  now.Add(-shiftTime),
	}
}

// arrive stellt ein Fahrzeug hinten an.
func (l *lane) arrive(at time.Time) {
	l.queue = append(l.queue, at)
	l.arrived++
}

// step bewegt die Warteschlange und liefert die Sensorflanken, die dabei entstehen.
func (l *lane) step(now time.Time, released bool) []controller.Input {
	if !l.atLine && len(l.queue) > 0 && !now.Before(l.lastLeft.Add(shiftTime)) {
		l.atLine = true
		l.since = now
	}
	if l.atLine && released && !now.Before(l.since.Add(crossingTime)) {
		wait := now.Sub(l.queue[0])
		l.queue = l.queue[1:]
		l.atLine = false
		l.lastLeft = now
		l.departed++
		l.waitSum += wait
		if wait > l.waitMax {
			l.waitMax = wait
		}
	}
	return l.edges(now)
}

// edges vergleicht den gewuenschten Sensorzustand mit dem gemeldeten und liefert die
// Aenderungen als Eingangsereignisse.
func (l *lane) edges(now time.Time) []controller.Input {
	var inputs []controller.Input
	for index := range l.pins {
		want := l.sensor(index)
		if want == l.occupied[index] {
			continue
		}
		l.occupied[index] = want
		inputs = append(inputs, controller.Input{Pin: l.pins[index], Active: want, Time: now})
	}
	return inputs
}

func (l *lane) sensor(index int) bool {
	if index == 0 {
		return l.atLine
	}
	return len(l.queue) > index
}

func (l *lane) waiting() int { return len(l.queue) }

// meanWait ist die wahre mittlere Wartezeit dieser Zufahrt, gemessen am Modell und nicht an
// den Sensoren. Sie dient als Gegenprobe zur Messung des Regelkreises.
func (l *lane) meanWait() time.Duration {
	if l.departed == 0 {
		return 0
	}
	return l.waitSum / time.Duration(l.departed)
}
