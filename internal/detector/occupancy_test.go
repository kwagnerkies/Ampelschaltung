package detector

import (
	"testing"
	"time"
)

var mapping = map[int]int{0: 0, 1: 1, 2: 3, 3: 6}

func TestReachSkipsGaps(t *testing.T) {
	o := NewOccupancy(3)
	o.Apply(2, true, base)

	if got := o.Reach(); got != 3 {
		t.Errorf("Reichweite %d, erwartet 3", got)
	}
	if o.AtStopLine() {
		t.Error("Haltelinie gilt als belegt")
	}
}

func TestOccupancySinceAndReset(t *testing.T) {
	o := NewOccupancy(2)
	o.Apply(1, true, base)
	if got := o.Since(1); got != base {
		t.Errorf("Zeitpunkt %s, erwartet %s", got, base)
	}

	o.Apply(1, true, base.Add(time.Second))
	if got := o.Since(1); got != base {
		t.Errorf("gleicher Pegel aenderte den Zeitpunkt auf %s", got)
	}

	o.Reset()
	if o.Reach() != 0 {
		t.Error("Reset liess Belegung stehen")
	}
}
