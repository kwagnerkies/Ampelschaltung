package steuerung

import (
	"ampel/src/signal"
	"time"
)

type Phase int

const (
	PhaseStartup Phase = iota
	PhaseNS
	PhaseEW
	PhaseFault
)

type Stage int

const (
	StageGreen Stage = iota
	StageYellow
	StageAllRed
	StageRedYellow
)

func (p Phase) String() string {
	switch p {
	case PhaseStartup:
		return "Start"
	case PhaseNS:
		return "NS"
	case PhaseEW:
		return "OW"
	case PhaseFault:
		return "Stoerung"
	}
	return "unbekannt"
}

func (s Stage) String() string {
	switch s {
	case StageGreen:
		return "Gruen"
	case StageYellow:
		return "Gelb"
	case StageAllRed:
		return "Allrot"
	case StageRedYellow:
		return "RotGelb"
	}
	return "unbekannt"
}

func (p Phase) Directions() []signal.Direction {
	switch p {
	case PhaseNS:
		return []signal.Direction{signal.North, signal.South}
	case PhaseEW:
		return []signal.Direction{signal.East, signal.West}
	}
	return nil
}

func PhaseOf(direction signal.Direction) Phase {
	switch direction {
	case signal.North, signal.South:
		return PhaseNS
	}
	return PhaseEW
}

func (p Phase) Other() Phase {
	switch p {
	case PhaseNS:
		return PhaseEW
	case PhaseEW:
		return PhaseNS
	}
	return PhaseNS
}

type State struct {
	Phase  Phase
	Stage  Stage
	Since  time.Time
	Target time.Duration
}

func (s State) Name() string {
	if s.Stage == StageAllRed {
		return "Allrot"
	}
	return s.Phase.String() + "_" + s.Stage.String()
}

func (s State) Aspects() [signal.DirectionCount]signal.Aspect {
	aspects := [signal.DirectionCount]signal.Aspect{
		signal.AspectRed, signal.AspectRed, signal.AspectRed, signal.AspectRed,
	}
	var aspect signal.Aspect
	switch s.Stage {
	case StageGreen:
		aspect = signal.AspectGreen
	case StageYellow:
		aspect = signal.AspectYellow
	case StageRedYellow:
		aspect = signal.AspectRedYellow
	default:
		return aspects
	}
	for _, direction := range s.Phase.Directions() {
		aspects[direction] = aspect
	}
	return aspects
}

type Timing struct {
	Yellow    time.Duration
	AllRed    time.Duration
	RedYellow time.Duration
}

func (t Timing) Intergreen() time.Duration { return t.Yellow + t.AllRed + t.RedYellow }

type Machine struct {
	timing Timing
	state  State
}

func NewMachine(timing Timing, now time.Time) *Machine {
	return &Machine{
		timing: timing,
		state:  State{Phase: PhaseStartup, Stage: StageAllRed, Since: now},
	}
}

func (m *Machine) State() State { return m.state }

func (m *Machine) SetTarget(target time.Duration) { m.state.Target = target }

func (m *Machine) Advance(now time.Time, endGreen bool) bool {
	elapsed := now.Sub(m.state.Since)
	switch m.state.Stage {
	case StageGreen:
		if !endGreen {
			return false
		}
		m.enter(now, m.state.Phase, StageYellow)
	case StageYellow:
		if elapsed < m.timing.Yellow {
			return false
		}
		m.enter(now, m.state.Phase, StageAllRed)
	case StageAllRed:
		if elapsed < m.timing.AllRed {
			return false
		}
		m.enter(now, m.state.Phase.Other(), StageRedYellow)
	case StageRedYellow:
		if elapsed < m.timing.RedYellow {
			return false
		}
		m.enter(now, m.state.Phase, StageGreen)
	}
	return true
}

func (m *Machine) Restart(now time.Time) {
	m.state = State{Phase: PhaseStartup, Stage: StageAllRed, Since: now}
}

func (m *Machine) Fault(now time.Time) {
	m.state = State{Phase: PhaseFault, Stage: StageAllRed, Since: now}
}

func (m *Machine) enter(now time.Time, phase Phase, stage Stage) {
	target := m.state.Target
	if stage != StageGreen {
		target = 0
	}
	m.state = State{Phase: phase, Stage: stage, Since: now, Target: target}
}
