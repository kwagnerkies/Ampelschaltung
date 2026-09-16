package strategy

import (
	"testing"
	"time"
)

var start = time.Date(2026, 4, 1, 7, 0, 0, 0, time.UTC)

func TestFixedHoldsConfiguredGreen(t *testing.T) {
	f := NewFixed(15 * time.Second)

	if got := f.TargetGreen(View{}); got != 15*time.Second {
		t.Errorf("Zielgruenzeit %s, erwartet 15s", got)
	}
	if f.Name() != "festzeit" {
		t.Errorf("Name %q", f.Name())
	}
}

// Die Festzeitsteuerung beendet die Freigabe genau nach der konfigurierten Zeit, unabhaengig
// von jeder Nachfrage.
func TestFixedEndsOnlyAfterGreenTime(t *testing.T) {
	f := NewFixed(15 * time.Second)
	cases := []struct {
		elapsed time.Duration
		want    bool
	}{
		{0, false},
		{14*time.Second + 999*time.Millisecond, false},
		{15 * time.Second, true},
		{time.Minute, true},
	}
	for _, tc := range cases {
		view := View{Now: start.Add(tc.elapsed), GreenSince: start, OtherQueue: 10, OtherOldestWait: time.Hour}
		if got := f.EndGreen(view); got != tc.want {
			t.Errorf("nach %s ergibt EndGreen %v, erwartet %v", tc.elapsed, got, tc.want)
		}
	}
}
