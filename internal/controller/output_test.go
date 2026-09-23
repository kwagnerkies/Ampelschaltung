package controller

import (
	"errors"
	"testing"

	"ampel/internal/hal"
	"ampel/internal/light"
)

func newOutput(t *testing.T) (*Output, *hal.Mock) {
	t.Helper()
	mock := hal.NewMock(LampCount, 1)
	return NewOutput(mock), mock
}

// Ein Konflikt darf die Hardware nicht erreichen. Das ist der Kern von Abschnitt 11 des Plans.
func TestShowRejectsConflictBeforeWriting(t *testing.T) {
	output, mock := newOutput(t)
	if err := output.Show([light.DirectionCount]light.Aspect{
		light.AspectRed, light.AspectRed, light.AspectRed, light.AspectRed}); err != nil {
		t.Fatalf("alles auf Rot: %v", err)
	}
	writes := mock.Writes()

	err := output.Show([light.DirectionCount]light.Aspect{
		light.AspectRedYellow, light.AspectRedYellow, light.AspectRed, light.AspectRed})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("Fehler %v, erwartet ErrConflict", err)
	}
	if mock.Writes() != writes {
		t.Error("der Konflikt wurde in die Hardware geschrieben")
	}
	for i, aspect := range output.Aspects() {
		if aspect != light.AspectRed {
			t.Errorf("Zufahrt %s zeigt %s, erwartet unveraendert Rot", light.Direction(i), aspect)
		}
	}
}

func TestShowRejectsSequenceViolation(t *testing.T) {
	output, mock := newOutput(t)
	if err := output.Show([light.DirectionCount]light.Aspect{
		light.AspectRed, light.AspectRed, light.AspectRed, light.AspectRed}); err != nil {
		t.Fatalf("alles auf Rot: %v", err)
	}
	writes := mock.Writes()

	// Gruen ohne RotGelb davor verletzt die Signalfolge, ist aber konfliktfrei.
	err := output.Show([light.DirectionCount]light.Aspect{
		light.AspectGreen, light.AspectRed, light.AspectGreen, light.AspectRed})
	if err == nil {
		t.Fatal("Gruen direkt nach Rot wurde angenommen")
	}
	if mock.Writes() != writes {
		t.Error("der unzulaessige Wechsel wurde in die Hardware geschrieben")
	}
}

func TestShowWritesPattern(t *testing.T) {
	output, mock := newOutput(t)
	steps := [][light.DirectionCount]light.Aspect{
		{light.AspectRed, light.AspectRed, light.AspectRed, light.AspectRed},
		{light.AspectRedYellow, light.AspectRed, light.AspectRedYellow, light.AspectRed},
		{light.AspectGreen, light.AspectRed, light.AspectGreen, light.AspectRed},
	}
	for _, step := range steps {
		if err := output.Show(step); err != nil {
			t.Fatalf("Show: %v", err)
		}
	}

	pattern := mock.Pattern()
	for _, bit := range []int{2, 8, 3, 9} {
		if !pattern[bit] {
			t.Errorf("bit %d leuchtet nicht", bit)
		}
	}
	if mock.Writes() != len(steps) {
		t.Errorf("%d Schreibzugriffe, erwartet %d", mock.Writes(), len(steps))
	}
}

func TestDarkTurnsEverythingOff(t *testing.T) {
	output, mock := newOutput(t)
	if err := output.Show([light.DirectionCount]light.Aspect{
		light.AspectRed, light.AspectRed, light.AspectRed, light.AspectRed}); err != nil {
		t.Fatalf("alles auf Rot: %v", err)
	}
	if err := output.Dark(); err != nil {
		t.Fatalf("Dark: %v", err)
	}
	for bit, lit := range mock.Pattern() {
		if lit {
			t.Errorf("bit %d leuchtet nach Dark", bit)
		}
	}
}
