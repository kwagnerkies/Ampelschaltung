package controller

import (
	"errors"
	"testing"

	"ampel/internal/light"
)

func allRed() [light.DirectionCount]light.Aspect {
	return [light.DirectionCount]light.Aspect{
		light.AspectRed, light.AspectRed, light.AspectRed, light.AspectRed,
	}
}

// Die Konfliktmatrix wird ueber alle Richtungspaare geprueft: gegenueberliegende Zufahrten
// duerfen gemeinsam gruen sein, kreuzende nie.
func TestCheckCoversAllDirectionPairs(t *testing.T) {
	for _, a := range light.Directions() {
		for _, b := range light.Directions() {
			if a == b {
				continue
			}
			aspects := allRed()
			aspects[a] = light.AspectGreen
			aspects[b] = light.AspectGreen

			opposite := (a == light.North && b == light.South) || (a == light.South && b == light.North) ||
				(a == light.East && b == light.West) || (a == light.West && b == light.East)
			err := Check(aspects)
			if opposite && err != nil {
				t.Errorf("%s und %s gemeinsam gruen wurde abgewiesen: %v", a, b, err)
			}
			if !opposite && err == nil {
				t.Errorf("%s und %s gemeinsam gruen wurde angenommen", a, b)
			}
			if !opposite && !errors.Is(err, ErrConflict) {
				t.Errorf("Fehler %v, erwartet ErrConflict", err)
			}
		}
	}
}

// Gelb gibt noch frei: in der Kreuzung stehen Fahrzeuge. Ein kreuzendes RotGelb darf deshalb
// nicht gleichzeitig auftreten.
func TestCheckRejectsOverlappingTransitions(t *testing.T) {
	cases := []struct {
		name    string
		aspects [light.DirectionCount]light.Aspect
	}{
		{"gelb gegen kreuzendes gruen", [light.DirectionCount]light.Aspect{
			light.AspectYellow, light.AspectGreen, light.AspectRed, light.AspectRed}},
		{"rotgelb gegen kreuzendes gelb", [light.DirectionCount]light.Aspect{
			light.AspectRed, light.AspectRedYellow, light.AspectYellow, light.AspectRed}},
		{"rotgelb gegen kreuzendes rotgelb", [light.DirectionCount]light.Aspect{
			light.AspectRedYellow, light.AspectRedYellow, light.AspectRed, light.AspectRed}},
		{"dunkler kopf neben freigabe", [light.DirectionCount]light.Aspect{
			light.AspectGreen, light.AspectOff, light.AspectRed, light.AspectRed}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := Check(tc.aspects); !errors.Is(err, ErrConflict) {
				t.Errorf("Zustand wurde mit %v angenommen, erwartet ErrConflict", err)
			}
		})
	}
}

func TestCheckAcceptsRegularStates(t *testing.T) {
	cases := []struct {
		name    string
		aspects [light.DirectionCount]light.Aspect
	}{
		{"alles rot", allRed()},
		{"nord und sued gruen", [light.DirectionCount]light.Aspect{
			light.AspectGreen, light.AspectRed, light.AspectGreen, light.AspectRed}},
		{"ost und west gruen", [light.DirectionCount]light.Aspect{
			light.AspectRed, light.AspectGreen, light.AspectRed, light.AspectGreen}},
		{"nord und sued gelb", [light.DirectionCount]light.Aspect{
			light.AspectYellow, light.AspectRed, light.AspectYellow, light.AspectRed}},
		{"ost und west rotgelb", [light.DirectionCount]light.Aspect{
			light.AspectRed, light.AspectRedYellow, light.AspectRed, light.AspectRedYellow}},
		{"notzustand blinkt", [light.DirectionCount]light.Aspect{
			light.AspectYellowFlash, light.AspectYellowFlash, light.AspectYellowFlash, light.AspectYellowFlash}},
		{"alles abgeschaltet", [light.DirectionCount]light.Aspect{
			light.AspectOff, light.AspectOff, light.AspectOff, light.AspectOff}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := Check(tc.aspects); err != nil {
				t.Errorf("zulaessiger Zustand wurde abgewiesen: %v", err)
			}
		})
	}
}

func TestConflicting(t *testing.T) {
	if Conflicting(light.North, light.South) || Conflicting(light.East, light.West) {
		t.Error("gegenueberliegende Zufahrten gelten als konfliktaer")
	}
	if !Conflicting(light.North, light.East) || !Conflicting(light.West, light.South) {
		t.Error("kreuzende Zufahrten gelten als vertraeglich")
	}
}
