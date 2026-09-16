package strategy

import (
	"fmt"
	"time"
)

// Following ist die verkehrsabhaengige Gruenzeitverlaengerung. Die Freigabe beginnt mit einer
// Grundzeit. Faehrt ein Fahrzeug dicht hinter seinem Vorgaenger ueber die Haltelinie, haengt
// das eine feste Verlaengerung an: zwei Fahrzeuge kurz hintereinander bedeuten, dass noch
// mehr kommt. Bei der Hoechstgruenzeit ist Schluss, sonst verhungert die andere Richtung.
//
// Die Strategie haelt keinen eigenen Zustand. Alles, was sie braucht, steht in der View,
// und damit ist jede Entscheidung fuer sich testbar.
type Following struct {
	base time.Duration
	step time.Duration
	max  time.Duration
}

var _ Strategy = (*Following)(nil)

func NewFollowing(base, step, max time.Duration) (*Following, error) {
	if base <= 0 {
		return nil, fmt.Errorf("grundgruenzeit %s", base)
	}
	if step <= 0 {
		return nil, fmt.Errorf("verlaengerung %s", step)
	}
	if max < base {
		return nil, fmt.Errorf("hoechstgruenzeit %s liegt unter der grundzeit %s", max, base)
	}
	return &Following{base: base, step: step, max: max}, nil
}

func (f *Following) Name() string { return "adaptiv" }

// TargetGreen ist die Grundzeit zuzueglich einer Verlaengerung je dicht folgendem Fahrzeug.
func (f *Following) TargetGreen(view View) time.Duration {
	target := f.base + time.Duration(view.Following)*f.step
	if target > f.max {
		return f.max
	}
	return target
}

func (f *Following) EndGreen(view View) bool { return view.Green() >= f.TargetGreen(view) }
