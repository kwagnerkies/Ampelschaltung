package mode

import "time"

type Switch struct {
	debounce time.Duration
	level    bool
	raw      bool
	rawAt    time.Time
}

func NewSwitch(initial bool, debounce time.Duration, now time.Time) *Switch {
	return &Switch{debounce: debounce, level: initial, raw: initial, rawAt: now}
}

func (s *Switch) Poll(level bool, now time.Time) bool {
	if level != s.raw {
		s.raw = level
		s.rawAt = now
		return false
	}
	if s.raw == s.level || now.Sub(s.rawAt) < s.debounce {
		return false
	}
	s.level = s.raw
	return true
}

func (s *Switch) Level() bool { return s.level }
