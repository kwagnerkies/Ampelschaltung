package strategy

import (
	"fmt"
	"time"
)

type View struct {
	Now        time.Time
	GreenSince time.Time
	Following  int
}

func (v View) Green() time.Duration { return v.Now.Sub(v.GreenSince) }

type Strategy interface {
	Name() string
	TargetGreen(view View) time.Duration
	EndGreen(view View) bool
}

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

func (f *Following) TargetGreen(view View) time.Duration {
	target := f.base + time.Duration(view.Following)*f.step
	if target > f.max {
		return f.max
	}
	return target
}

func (f *Following) EndGreen(view View) bool { return view.Green() >= f.TargetGreen(view) }
