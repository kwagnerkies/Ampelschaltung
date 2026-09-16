package light

import "testing"

// Rot und Gelb leuchten in der deutschen Signalfolge gemeinsam. Der Treiber muss das
// koennen, deshalb steht es hier ausdruecklich.
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

// Der Notzustand und das Abschalten muessen aus jedem Signalbild erreichbar sein.
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
