// Paket clock kapselt die Zeit, damit die Regelungslogik ohne echtes Warten testbar bleibt.
package clock

import "time"

type Ticker interface {
	C() <-chan time.Time
	Stop()
}

type Clock interface {
	Now() time.Time
	After(d time.Duration) <-chan time.Time
	Ticker(d time.Duration) Ticker
}
