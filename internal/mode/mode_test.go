package mode

import (
	"testing"
	"time"
)

var start = time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)

// Ein prellender Kippschalter darf keinen Moduswechsel erzeugen.
func TestSwitchIgnoresBouncing(t *testing.T) {
	s := NewSwitch(false, 100*time.Millisecond, start)

	level := true
	now := start
	for i := 0; i < 9; i++ {
		if s.Poll(level, now) {
			t.Fatalf("Wechsel nach %s Prellen", now.Sub(start))
		}
		level = !level
		now = now.Add(20 * time.Millisecond)
	}

	// Danach liegt der Pegel ruhig an.
	now = now.Add(20 * time.Millisecond)
	if s.Poll(true, now) {
		t.Fatal("Wechsel schon beim ersten ruhigen Abfragen")
	}
	now = now.Add(100 * time.Millisecond)
	if !s.Poll(true, now) {
		t.Fatal("kein Wechsel nach der Ruhezeit")
	}
	if !s.Level() {
		t.Error("der uebernommene Pegel ist falsch")
	}
}

func TestSwitchReportsChangeOnce(t *testing.T) {
	s := NewSwitch(false, 100*time.Millisecond, start)
	now := start.Add(time.Second)

	s.Poll(true, now)
	now = now.Add(150 * time.Millisecond)
	if !s.Poll(true, now) {
		t.Fatal("kein Wechsel nach der Ruhezeit")
	}
	now = now.Add(time.Second)
	if s.Poll(true, now) {
		t.Error("der Wechsel wurde zweimal gemeldet")
	}
}

// Der Reset loest erst nach zwei Sekunden Dauerdruck aus, und nur einmal je Druck.
func TestButtonNeedsLongPress(t *testing.T) {
	b := NewButton(2 * time.Second)
	now := start

	if b.Poll(true, now) {
		t.Fatal("der Taster loeste sofort aus")
	}
	now = now.Add(1900 * time.Millisecond)
	if b.Poll(true, now) {
		t.Fatalf("der Taster loeste nach %s aus", b.Held(now))
	}
	now = now.Add(200 * time.Millisecond)
	if !b.Poll(true, now) {
		t.Fatal("der Taster loeste nach zwei Sekunden nicht aus")
	}
	now = now.Add(time.Second)
	if b.Poll(true, now) {
		t.Error("der Taster loeste zweimal beim selben Druck aus")
	}

	// Nach dem Loslassen ist wieder ein Ausloesen moeglich.
	if b.Poll(false, now) {
		t.Fatal("das Loslassen loeste aus")
	}
	now = now.Add(100 * time.Millisecond)
	b.Poll(true, now)
	now = now.Add(2 * time.Second)
	if !b.Poll(true, now) {
		t.Error("der zweite Druck loeste nicht aus")
	}
}

func TestButtonForgetsShortPress(t *testing.T) {
	b := NewButton(2 * time.Second)
	now := start

	b.Poll(true, now)
	now = now.Add(500 * time.Millisecond)
	b.Poll(false, now)
	now = now.Add(500 * time.Millisecond)
	b.Poll(true, now)
	now = now.Add(1600 * time.Millisecond)

	if b.Poll(true, now) {
		t.Error("zwei kurze Druecke wurden zu einem langen zusammengefasst")
	}
}
