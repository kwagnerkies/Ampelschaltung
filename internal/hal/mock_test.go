package hal

import (
	"testing"
	"time"
)

func TestMockPatternIsACopy(t *testing.T) {
	m := NewMock(4, 1)
	if err := m.Write([]bool{true, false, true, false}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	pattern := m.Pattern()
	pattern[0] = false
	if !m.Pattern()[0] {
		t.Error("Pattern gibt den inneren Zustand heraus")
	}
	if m.Writes() != 1 {
		t.Errorf("Schreibzugriffe %d, erwartet 1", m.Writes())
	}
}

func TestMockEmitDeliversEventAndLevel(t *testing.T) {
	m := NewMock(4, 1)
	at := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)

	if err := m.Emit(5, true, at); err != nil {
		t.Fatalf("Emit: %v", err)
	}
	event := <-m.Events()
	if event.Pin != 5 || !event.Active || event.Time != at {
		t.Errorf("Ereignis %+v, erwartet Pin 5, aktiv, %s", event, at)
	}
	active, err := m.Read(5)
	if err != nil || !active {
		t.Errorf("Read ergibt %v, %v, erwartet true, nil", active, err)
	}
}

// Ein voller Kanal darf den Test nicht blockieren, sondern muss auffallen.
func TestMockEmitReportsFullChannel(t *testing.T) {
	m := NewMock(4, 1)
	at := time.Now()
	if err := m.Emit(5, true, at); err != nil {
		t.Fatalf("erste Flanke: %v", err)
	}
	if err := m.Emit(6, true, at); err == nil {
		t.Error("zweite Flanke wurde trotz vollem Kanal angenommen")
	}
}
