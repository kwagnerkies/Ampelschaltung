package main

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"ampel/internal/config"
	"ampel/internal/controller"
	"ampel/internal/light"
)

type step struct {
	aspects [light.DirectionCount]light.Aspect
	hold    time.Duration
}

// showSequence fuehrt beide Freigabephasen einmal durch. Damit ist die deutsche Signalfolge
// an allen vier Koepfen sichtbar, und der Weg ueber Sicherheitspruefung, Bus und Treiber ist
// vollstaendig geprueft.
func showSequence(ctx context.Context, out io.Writer, output *controller.Output, timing config.Timing) error {
	for _, s := range sequenceSteps(timing) {
		if err := output.Show(s.aspects); err != nil {
			return fmt.Errorf("signalbild schalten: %w", err)
		}
		fmt.Fprintf(out, "  %5s  %s\n", s.hold, describe(s.aspects))
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(s.hold):
		}
	}
	return nil
}

func sequenceSteps(timing config.Timing) []step {
	const (
		ns = 0
		ew = 1
	)
	phase := func(free int, aspect light.Aspect) [light.DirectionCount]light.Aspect {
		aspects := [light.DirectionCount]light.Aspect{
			light.AspectRed, light.AspectRed, light.AspectRed, light.AspectRed,
		}
		if free == ns {
			aspects[light.North], aspects[light.South] = aspect, aspect
		} else {
			aspects[light.East], aspects[light.West] = aspect, aspect
		}
		return aspects
	}
	allRed := phase(ns, light.AspectRed)

	steps := []step{{aspects: allRed, hold: timing.AllRed.Duration()}}
	for _, free := range []int{ns, ew} {
		steps = append(steps,
			step{phase(free, light.AspectRedYellow), timing.RedYellow.Duration()},
			step{phase(free, light.AspectGreen), timing.BaseGreen.Duration()},
			step{phase(free, light.AspectYellow), timing.Yellow.Duration()},
			step{allRed, timing.AllRed.Duration()},
		)
	}
	return steps
}

func describe(aspects [light.DirectionCount]light.Aspect) string {
	parts := make([]string, 0, light.DirectionCount)
	for i, aspect := range aspects {
		parts = append(parts, fmt.Sprintf("%s %s", light.Direction(i), aspect))
	}
	return strings.Join(parts, ", ")
}
