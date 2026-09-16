package main

import (
	"math"
	"math/rand"
	"time"

	"ampel/internal/light"
)

// arrivals erzeugt Fahrzeugankuenfte je Zufahrt als Poisson-Prozess. Die Rate folgt einem
// Tagesgang, damit Nord und Sued morgens und Ost und West abends stark belastet sind.
type arrivals struct {
	rng       *rand.Rand
	rates     [light.DirectionCount]float64
	peaks     [light.DirectionCount]float64
	amplitude float64
	next      [light.DirectionCount]time.Time
}

// peakHours sind die Stunden der Tagesspitze je Zufahrt.
var peakHours = [light.DirectionCount]float64{8, 17, 8, 17}

// newArrivals erwartet die Grundrate in Fahrzeugen pro Sekunde und die Staerke des
// Tagesgangs zwischen null und eins.
func newArrivals(seed int64, rates [light.DirectionCount]float64, amplitude float64, start time.Time) *arrivals {
	a := &arrivals{
		rng:       rand.New(rand.NewSource(seed)),
		rates:     rates,
		peaks:     peakHours,
		amplitude: amplitude,
	}
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
	rate := a.rate(direction, from)
	if rate <= 0 {
		return from.Add(24 * time.Hour)
	}
	gap := -math.Log(1-a.rng.Float64()) / rate
	return from.Add(time.Duration(gap * float64(time.Second)))
}

// rate ist die Ankunftsrate zur Tageszeit von at.
func (a *arrivals) rate(direction light.Direction, at time.Time) float64 {
	base := a.rates[direction]
	if base <= 0 || a.amplitude <= 0 {
		return base
	}
	hour := float64(at.Hour()) + float64(at.Minute())/60
	factor := 1 + a.amplitude*math.Cos(2*math.Pi*(hour-a.peaks[direction])/24)
	if factor < 0 {
		factor = 0
	}
	return base * factor
}
