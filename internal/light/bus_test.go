package light

import (
	"reflect"
	"testing"
)

// Startbelegung aus dem Plan: je Kopf drei aufeinanderfolgende Bits, vier freie Positionen.
var planPositions = [DirectionCount][3]int{
	North: {0, 1, 2},
	East:  {3, 4, 5},
	South: {6, 7, 8},
	West:  {9, 10, 11},
}

func TestPatternAllRed(t *testing.T) {
	bus, err := NewBus(planPositions, 16)
	if err != nil {
		t.Fatalf("NewBus: %v", err)
	}
	pattern := bus.Pattern([DirectionCount]Aspect{AspectRed, AspectRed, AspectRed, AspectRed})

	want := make([]bool, 16)
	for _, bit := range []int{0, 3, 6, 9} {
		want[bit] = true
	}
	if !reflect.DeepEqual(pattern, want) {
		t.Errorf("Muster %v, erwartet %v", pattern, want)
	}
}

// Bei Rot und Gelb muessen zwei Bits eines Kopfes gleichzeitig gesetzt sein.
func TestPatternRedYellowLightsTwoBits(t *testing.T) {
	bus, err := NewBus(planPositions, 16)
	if err != nil {
		t.Fatalf("NewBus: %v", err)
	}
	pattern := bus.Pattern([DirectionCount]Aspect{AspectRedYellow, AspectRed, AspectRedYellow, AspectRed})

	for _, bit := range []int{0, 1, 6, 7, 3, 9} {
		if !pattern[bit] {
			t.Errorf("bit %d leuchtet nicht", bit)
		}
	}
	for _, bit := range []int{2, 4, 5, 8, 10, 11, 12, 13, 14, 15} {
		if pattern[bit] {
			t.Errorf("bit %d leuchtet, erwartet dunkel", bit)
		}
	}
}

func TestPatternPhaseNorthSouthGreen(t *testing.T) {
	bus, err := NewBus(planPositions, 16)
	if err != nil {
		t.Fatalf("NewBus: %v", err)
	}
	pattern := bus.Pattern([DirectionCount]Aspect{AspectGreen, AspectRed, AspectGreen, AspectRed})

	on := []int{}
	for bit, lit := range pattern {
		if lit {
			on = append(on, bit)
		}
	}
	if want := []int{2, 3, 8, 9}; !reflect.DeepEqual(on, want) {
		t.Errorf("es leuchten die Bits %v, erwartet %v", on, want)
	}
}

func TestNewBusRejectsBadPositions(t *testing.T) {
	cases := []struct {
		name      string
		positions [DirectionCount][3]int
		bits      int
	}{
		{"position ausserhalb der kette", planPositions, 8},
		{"negative position", [DirectionCount][3]int{North: {-1, 1, 2}, East: {3, 4, 5}, South: {6, 7, 8}, West: {9, 10, 11}}, 16},
		{"position doppelt belegt", [DirectionCount][3]int{North: {0, 1, 2}, East: {2, 4, 5}, South: {6, 7, 8}, West: {9, 10, 11}}, 16},
		{"kette ohne bits", planPositions, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := NewBus(tc.positions, tc.bits); err == nil {
				t.Error("Belegung wurde angenommen")
			}
		})
	}
}
