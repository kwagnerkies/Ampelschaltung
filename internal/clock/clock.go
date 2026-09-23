package clock

import (
	"sync"
	"time"
)

type Ticker interface {
	C() <-chan time.Time
	Stop()
}

type Clock interface {
	Now() time.Time
	After(d time.Duration) <-chan time.Time
	Ticker(d time.Duration) Ticker
}

type Real struct{}

var _ Clock = Real{}

func NewReal() Real { return Real{} }

func (Real) Now() time.Time { return time.Now() }

func (Real) After(d time.Duration) <-chan time.Time { return time.After(d) }

func (Real) Ticker(d time.Duration) Ticker { return realTicker{ticker: time.NewTicker(d)} }

type realTicker struct {
	ticker *time.Ticker
}

func (r realTicker) C() <-chan time.Time { return r.ticker.C }

func (r realTicker) Stop() { r.ticker.Stop() }

type Fake struct {
	mu      sync.Mutex
	now     time.Time
	waiters []*waiter
}

type waiter struct {
	deadline time.Time
	period   time.Duration
	ch       chan time.Time
	stopped  bool
}

var _ Clock = (*Fake)(nil)

func NewFake(start time.Time) *Fake { return &Fake{now: start} }

func (f *Fake) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.now
}

func (f *Fake) After(d time.Duration) <-chan time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.add(d, 0).ch
}

func (f *Fake) Ticker(d time.Duration) Ticker {
	if d <= 0 {
		panic("clock: Ticker mit nicht positiver Periode")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return &fakeTicker{fake: f, waiter: f.add(d, d)}
}

func (f *Fake) Advance(d time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	target := f.now.Add(d)
	for {
		due := f.earliest(target)
		if due == nil {
			break
		}
		f.now = due.deadline
		select {
		case due.ch <- f.now:
		default:
		}
		if due.period > 0 {
			due.deadline = due.deadline.Add(due.period)
			continue
		}
		f.remove(due)
	}
	f.now = target
}

func (f *Fake) add(d, period time.Duration) *waiter {
	w := &waiter{
		deadline: f.now.Add(d),
		period:   period,
		ch:       make(chan time.Time, 1),
	}
	f.waiters = append(f.waiters, w)
	return w
}

func (f *Fake) earliest(target time.Time) *waiter {
	var best *waiter
	for _, w := range f.waiters {
		if w.stopped || w.deadline.After(target) {
			continue
		}
		if best == nil || w.deadline.Before(best.deadline) {
			best = w
		}
	}
	return best
}

func (f *Fake) remove(target *waiter) {
	for i, w := range f.waiters {
		if w == target {
			f.waiters = append(f.waiters[:i], f.waiters[i+1:]...)
			return
		}
	}
}

type fakeTicker struct {
	fake   *Fake
	waiter *waiter
}

func (t *fakeTicker) C() <-chan time.Time { return t.waiter.ch }

func (t *fakeTicker) Stop() {
	t.fake.mu.Lock()
	defer t.fake.mu.Unlock()
	t.waiter.stopped = true
	t.fake.remove(t.waiter)
}
