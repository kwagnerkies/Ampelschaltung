package clock

import (
	"testing"
	"time"
)

var start = time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)

func TestFakeAfterFiresAtDeadline(t *testing.T) {
	f := NewFake(start)
	ch := f.After(10 * time.Millisecond)

	f.Advance(9 * time.Millisecond)
	if got := receive(ch); !got.IsZero() {
		t.Fatalf("Timer feuerte zu frueh bei %s", got)
	}

	f.Advance(time.Millisecond)
	got := receive(ch)
	if want := start.Add(10 * time.Millisecond); got != want {
		t.Errorf("Zeitstempel %s, erwartet %s", got, want)
	}
	if now := f.Now(); now != start.Add(10*time.Millisecond) {
		t.Errorf("Uhr steht auf %s, erwartet %s", now, start.Add(10*time.Millisecond))
	}
}

func TestFakeAdvanceFiresInOrder(t *testing.T) {
	f := NewFake(start)
	late := f.After(10 * time.Millisecond)
	early := f.After(4 * time.Millisecond)

	f.Advance(20 * time.Millisecond)

	if got, want := receive(early), start.Add(4*time.Millisecond); got != want {
		t.Errorf("frueher Timer bei %s, erwartet %s", got, want)
	}
	if got, want := receive(late), start.Add(10*time.Millisecond); got != want {
		t.Errorf("spaeter Timer bei %s, erwartet %s", got, want)
	}
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

func TestFakeAfterDoesNotFireTwice(t *testing.T) {
	f := NewFake(start)
	ch := f.After(time.Millisecond)

	f.Advance(time.Millisecond)
	receive(ch)
	f.Advance(time.Second)
	if got := receive(ch); !got.IsZero() {
		t.Errorf("Einmaltimer feuerte erneut bei %s", got)
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
