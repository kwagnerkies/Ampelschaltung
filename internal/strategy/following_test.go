package strategy

import (
	"testing"
	"time"
)

var start = time.Date(2026, 4, 1, 7, 0, 0, 0, time.UTC)

func following(t *testing.T) *Following {
	t.Helper()
	f, err := NewFollowing(8*time.Second, 3*time.Second, 30*time.Second)
	if err != nil {
		t.Fatalf("NewFollowing: %v", err)
	}
	return f
}

// Zwei dicht aufeinander folgende Fahrzeuge verlaengern um drei Sekunden, jedes weitere um
// drei weitere.
func TestEachFollowingVehicleExtends(t *testing.T) {
	f := following(t)
	cases := map[int]time.Duration{
		0: 8 * time.Second,
		1: 11 * time.Second,
		2: 14 * time.Second,
		5: 23 * time.Second,
	}
	for count, want := range cases {
		if got := f.TargetGreen(View{Following: count}); got != want {
			t.Errorf("%d folgende Fahrzeuge ergeben %s, erwartet %s", count, got, want)
		}
	}
}

// Die Hoechstgruenzeit ist die Grenze, sonst verhungert die andere Richtung.
func TestExtensionStopsAtMaximum(t *testing.T) {
	f := following(t)
	if got := f.TargetGreen(View{Following: 100}); got != 30*time.Second {
		t.Errorf("Zielzeit %s, erwartet die Hoechstgruenzeit 30s", got)
	}
}

// Die Freigabe endet, sobald die Zielzeit erreicht ist, und keine Sekunde frueher.
func TestGreenEndsAtTarget(t *testing.T) {
	f := following(t)
	cases := []struct {
		elapsed   time.Duration
		following int
		want      bool
	}{
		{7 * time.Second, 0, false},
		{8 * time.Second, 0, true},
		{8 * time.Second, 1, false},
		{11 * time.Second, 1, true},
	}
	for _, tc := range cases {
		view := View{Now: start.Add(tc.elapsed), GreenSince: start, Following: tc.following}
		if got := f.EndGreen(view); got != tc.want {
			t.Errorf("nach %s mit %d folgenden Fahrzeugen ergibt EndGreen %v, erwartet %v",
				tc.elapsed, tc.following, got, tc.want)
		}
	}
}

func TestInvalidParameters(t *testing.T) {
	cases := map[string][3]time.Duration{
		"ohne Grundzeit":       {0, 3 * time.Second, 30 * time.Second},
		"ohne Verlaengerung":   {8 * time.Second, 0, 30 * time.Second},
		"Hoechstzeit zu klein": {8 * time.Second, 3 * time.Second, 5 * time.Second},
	}
	for name, values := range cases {
		if _, err := NewFollowing(values[0], values[1], values[2]); err == nil {
			t.Errorf("%s wurde angenommen", name)
		}
	}
}
