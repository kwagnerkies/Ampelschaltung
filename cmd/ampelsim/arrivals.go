package main

import (
	"math"
	"math/rand"
	"time"

	"ampel/internal/light"
)

// arrivals erzeugt Fahrzeugankuenfte je Zufahrt als Poisson-Prozess: die Abstaende sind
// exponentialverteilt, die Rate ist konstant.
type arrivals struct {
	rng   *rand.Rand
	rates [light.DirectionCount]float64
	next  [light.DirectionCount]time.Time
}

// newArrivals erwartet die Rate in Fahrzeugen pro Sekunde je Zufahrt.
func newArrivals(seed int64, rates [light.DirectionCount]float64, start time.Time) *arrivals {
	a := &arrivals{rng: rand.New(rand.NewSource(seed)), rates: rates}
	for _, direction := range light.Directions() {
		a.next[direction] = a.schedule(direction, start)
	}
	return a
}

// due liefert die Zufahrten, an denen jetzt ein Fahrzeug ankommt.
func (a *arrivals) due(now time.Time) []light.Direction {
	var due []light.Direction
	for _, direction := range light.Directions() {
		for !now.Before(a.next[direction]) {
			due = append(due, direction)
			a.next[direction] = a.schedule(direction, a.next[direction])
		}
	}
	return due
}

// schedule zieht den Abstand zur naechsten Ankunft aus der Exponentialverteilung.
func (a *arrivals) schedule(direction light.Direction, from time.Time) time.Time {
	rate := a.rates[direction]
	if rate <= 0 {
		return from.Add(24 * time.Hour)
	}
	gap := -math.Log(1-a.rng.Float64()) / rate
	return from.Add(time.Duration(gap * float64(time.Second)))
}
