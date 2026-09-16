package clock

import "time"

// Real ist die Uhr des Betriebssystems.
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
