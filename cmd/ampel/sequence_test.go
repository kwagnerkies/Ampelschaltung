package main

import (
	"testing"
	"time"

	"ampel/internal/config"
	"ampel/internal/hal"
	"ampel/internal/light"
)

// Die vorgefuehrte Folge muss den vollstaendigen Ausgabepfad ueberstehen: Konfliktmatrix,
// deutsche Signalfolge und Bitabbildung. Ein Fehler hier waere auf der Hardware sichtbar.
func TestSequencePassesGuardedOutput(t *testing.T) {
	cfg := config.Default()
	mock := hal.NewMock(len(cfg.Hardware.ShiftRegister.BitOrder), 1)
	output, err := newOutput(mock, &cfg)
	if err != nil {
		t.Fatalf("newOutput: %v", err)
	}

	greens := map[light.Direction]int{}
	for i, s := range sequenceSteps(cfg.Timing) {
		if err := output.Show(s.aspects); err != nil {
			t.Fatalf("schritt %d (%s): %v", i, describe(s.aspects), err)
		}
		for direction, aspect := range s.aspects {
			if aspect == light.AspectGreen {
				greens[light.Direction(direction)]++
			}
		}
	}

	for _, direction := range light.Directions() {
		if greens[direction] != 1 {
			t.Errorf("zufahrt %s war %d mal gruen, erwartet einmal", direction, greens[direction])
		}
	}
}

// Jede Freigabe beginnt mit Rot und Gelb und endet mit Gelb. Der Test liest die Folge je
// Zufahrt aus den Schritten heraus, unabhaengig von der Reihenfolge der Phasen.
func TestSequenceWrapsGreenInTransitions(t *testing.T) {
	cfg := config.Default()
	steps := sequenceSteps(cfg.Timing)

	for _, direction := range light.Directions() {
		var seen []light.Aspect
		for _, s := range steps {
			aspect := s.aspects[direction]
			if len(seen) == 0 || seen[len(seen)-1] != aspect {
				seen = append(seen, aspect)
			}
		}
		green := -1
		for i, aspect := range seen {
			if aspect == light.AspectGreen {
				green = i
			}
		}
		if green <= 0 || green+1 >= len(seen) {
			t.Fatalf("zufahrt %s: gruen liegt nicht innerhalb der folge %v", direction, seen)
		}
		if seen[green-1] != light.AspectRedYellow {
			t.Errorf("zufahrt %s zeigt vor Gruen %s, erwartet RotGelb", direction, seen[green-1])
		}
		if seen[green+1] != light.AspectYellow {
			t.Errorf("zufahrt %s zeigt nach Gruen %s, erwartet Gelb", direction, seen[green+1])
		}
	}
}

func TestSequenceHoldsComeFromConfig(t *testing.T) {
	cfg := config.Default()
	cfg.Timing.BaseGreen = config.Millis(4 * time.Second)
	steps := sequenceSteps(cfg.Timing)

	for _, s := range steps {
		var want time.Duration
		switch {
		case containsAspect(s.aspects, light.AspectGreen):
			want = 4 * time.Second
		case containsAspect(s.aspects, light.AspectYellow):
			want = cfg.Timing.Yellow.Duration()
		case containsAspect(s.aspects, light.AspectRedYellow):
			want = cfg.Timing.RedYellow.Duration()
		default:
			want = cfg.Timing.AllRed.Duration()
		}
		if s.hold != want {
			t.Errorf("%s haelt %s, erwartet %s", describe(s.aspects), s.hold, want)
		}
	}
}

func containsAspect(aspects [light.DirectionCount]light.Aspect, want light.Aspect) bool {
	for _, aspect := range aspects {
		if aspect == want {
			return true
		}
	}
	return false
}
