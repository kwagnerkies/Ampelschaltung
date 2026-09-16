package controller

import "time"

// DefaultWatchdog ist die groesste erlaubte Pause zwischen zwei Takten des Regelkreises.
const DefaultWatchdog = 500 * time.Millisecond

// Watchdog ueberwacht den Takt des Regelkreises. Er sitzt bewusst im Regelkreis selbst: der
// Steuerzustand gehoert einer einzigen Goroutine, ein zweiter Waechter duerfte ihn nicht in
// den Notzustand zwingen. Die Ueberschreitung faellt deshalb auf, sobald der Takt wieder
// laeuft.
type Watchdog struct {
	limit time.Duration
	last  time.Time
}

func NewWatchdog(limit time.Duration, now time.Time) *Watchdog {
	if limit <= 0 {
		limit = DefaultWatchdog
	}
	return &Watchdog{limit: limit, last: now}
}

func (w *Watchdog) Limit() time.Duration { return w.limit }

// Kick meldet einen Takt. Der Rueckgabewert ist der Abstand zum vorigen Takt und ob dieser
// zu gross war. Ein Ruecksprung der Uhr gilt nicht als Ueberschreitung.
func (w *Watchdog) Kick(now time.Time) (time.Duration, bool) {
	gap := now.Sub(w.last)
	w.last = now
	return gap, gap > w.limit
}
