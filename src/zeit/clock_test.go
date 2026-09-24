package zeit

import (
	"testing"
	"time"
)

var start = time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)

func TestFakeAdvanceMovesTheClock(t *testing.T) {
	f := NewFake(start)
	f.Advance(20 * time.Millisecond)
	if now, want := f.Now(), start.Add(20*time.Millisecond); now != want {
		t.Errorf("Uhr steht auf %s, erwartet %s", now, want)
	}
}

func TestFakeTickerRepeatsAndStops(t *testing.T) {
	f := NewFake(start)
	ticker := f.Ticker(5 * time.Millisecond)

	for i := 1; i <= 3; i++ {
		f.Advance(5 * time.Millisecond)
		want := start.Add(time.Duration(i) * 5 * time.Millisecond)
		if got := receive(ticker.C()); got != want {
			t.Fatalf("Tick %d bei %s, erwartet %s", i, got, want)
		}
	}

	ticker.Stop()
	f.Advance(20 * time.Millisecond)
	if got := receive(ticker.C()); !got.IsZero() {
		t.Errorf("Ticker feuerte nach Stop bei %s", got)
	}
}

func receive(ch <-chan time.Time) time.Time {
	select {
	case t := <-ch:
		return t
	default:
		return time.Time{}
	}
}
