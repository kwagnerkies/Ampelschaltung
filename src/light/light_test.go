package light

import (
	"testing"
)

func TestLampsPerAspect(t *testing.T) {
	cases := []struct {
		aspect Aspect
		want   Lamps
	}{
		{AspectRed, Lamps{Red: true}},
		{AspectRedYellow, Lamps{Red: true, Yellow: true}},
		{AspectGreen, Lamps{Green: true}},
		{AspectYellow, Lamps{Yellow: true}},
		{AspectYellowFlash, Lamps{Yellow: true}},
		{AspectOff, Lamps{}},
	}
	for _, tc := range cases {
		if got := tc.aspect.Lamps(); got != tc.want {
			t.Errorf("%s leuchtet %+v, erwartet %+v", tc.aspect, got, tc.want)
		}
	}
}

func TestZeroAspectIsRed(t *testing.T) {
	var a Aspect
	if a != AspectRed {
		t.Fatalf("Nullwert ist %s, erwartet Rot", a)
	}
}

func TestCanFollowGermanSequence(t *testing.T) {
	cases := []struct {
		previous Aspect
		next     Aspect
		want     bool
	}{
		{AspectRed, AspectRedYellow, true},
		{AspectRedYellow, AspectGreen, true},
		{AspectGreen, AspectYellow, true},
		{AspectYellow, AspectRed, true},
		{AspectOff, AspectRed, true},
		{AspectRed, AspectRed, true},

		{AspectRed, AspectGreen, false},
		{AspectGreen, AspectRed, false},
		{AspectRedYellow, AspectRed, true},
		{AspectYellow, AspectGreen, false},
		{AspectOff, AspectGreen, false},
		{AspectRedYellow, AspectYellow, false},
	}
	for _, tc := range cases {
		if got := tc.next.CanFollow(tc.previous); got != tc.want {
			t.Errorf("%s nach %s ergibt %v, erwartet %v", tc.next, tc.previous, got, tc.want)
		}
	}
}

func TestFaultAndOffReachableFromEverywhere(t *testing.T) {
	for _, previous := range []Aspect{AspectRed, AspectRedYellow, AspectGreen, AspectYellow, AspectOff, AspectYellowFlash} {
		if !AspectYellowFlash.CanFollow(previous) {
			t.Errorf("GelbBlinken ist nach %s nicht erreichbar", previous)
		}
		if !AspectOff.CanFollow(previous) {
			t.Errorf("Aus ist nach %s nicht erreichbar", previous)
		}
	}
}

func TestReleasing(t *testing.T) {
	releasing := map[Aspect]bool{
		AspectRedYellow:   true,
		AspectGreen:       true,
		AspectYellow:      true,
		AspectRed:         false,
		AspectOff:         false,
		AspectYellowFlash: false,
	}
	for aspect, want := range releasing {
		if got := aspect.Releasing(); got != want {
			t.Errorf("%s gibt frei %v, erwartet %v", aspect, got, want)
		}
	}
}

func TestHeadRunsGermanSequence(t *testing.T) {
	head := NewHead(North)
	sequence := []Aspect{AspectRed, AspectRedYellow, AspectGreen, AspectYellow, AspectRed}

	for _, aspect := range sequence {
		if err := head.Set(aspect); err != nil {
			t.Fatalf("Set(%s): %v", aspect, err)
		}
		if head.Aspect() != aspect {
			t.Fatalf("Kopf zeigt %s, erwartet %s", head.Aspect(), aspect)
		}
	}
}

func TestHeadRejectsGreenAfterRed(t *testing.T) {
	head := NewHead(East)
	if err := head.Set(AspectRed); err != nil {
		t.Fatalf("Set(Rot): %v", err)
	}
	if err := head.Set(AspectGreen); err == nil {
		t.Fatal("Gruen direkt nach Rot wurde angenommen")
	}
	if head.Aspect() != AspectRed {
		t.Errorf("Kopf zeigt %s, erwartet unveraendert Rot", head.Aspect())
	}
}

func TestHeadsStartOff(t *testing.T) {
	heads := NewHeads()
	for i, aspect := range heads.Aspects() {
		if aspect != AspectOff {
			t.Errorf("Zufahrt %s startet mit %s, erwartet Aus", Direction(i), aspect)
		}
	}
	for i, head := range heads {
		if head.Direction() != Direction(i) {
			t.Errorf("Kopf %d gehoert zu %s", i, head.Direction())
		}
	}
}

func TestHeadsSetIsAllOrNothing(t *testing.T) {
	heads := NewHeads()
	if err := heads.Set([DirectionCount]Aspect{AspectRed, AspectRed, AspectRed, AspectRed}); err != nil {
		t.Fatalf("alles auf Rot: %v", err)
	}

	err := heads.Set([DirectionCount]Aspect{AspectRedYellow, AspectRedYellow, AspectGreen, AspectRed})
	if err == nil {
		t.Fatal("Gruen direkt nach Rot wurde angenommen")
	}
	for i, aspect := range heads.Aspects() {
		if aspect != AspectRed {
			t.Errorf("Zufahrt %s zeigt %s, erwartet unveraendert Rot", Direction(i), aspect)
		}
	}
}
