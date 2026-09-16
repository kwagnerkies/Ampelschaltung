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
