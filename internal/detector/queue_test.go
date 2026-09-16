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

// Liegt S1 in einer Luecke zwischen zwei Autos, reicht der Stau trotzdem bis S2. Genau das
// unterscheidet die Belegungsmessung von einer Zaehlschranke.
func TestEstimateUsesReachNotCount(t *testing.T) {
	q, err := NewQueue(mapping, 3)
	if err != nil {
		t.Fatalf("NewQueue: %v", err)
	}
	o := NewOccupancy(3)
	o.Apply(0, true, base)
	o.Apply(2, true, base)

	if got := q.Estimate(o); got != 6 {
		t.Errorf("Rueckstau %d Fahrzeuge, erwartet 6", got)
	}
}

func TestEstimatePerReach(t *testing.T) {
	q, err := NewQueue(mapping, 3)
	if err != nil {
		t.Fatalf("NewQueue: %v", err)
	}
	o := NewOccupancy(3)
	want := []int{0, 1, 3, 6}
	for index := 0; index < 3; index++ {
		if got := q.Estimate(o); got != want[index] {
			t.Errorf("bei Reichweite %d ergibt die Tabelle %d, erwartet %d", index, got, want[index])
		}
		o.Apply(index, true, base)
	}
	if got := q.Estimate(o); got != 6 {
		t.Errorf("bei voller Belegung %d, erwartet 6", got)
	}
}

func TestNewQueueRejectsIncompleteTable(t *testing.T) {
	if _, err := NewQueue(map[int]int{0: 0, 1: 1}, 3); err == nil {
		t.Error("unvollstaendige Tabelle wurde angenommen")
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
