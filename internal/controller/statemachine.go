package controller

import "time"

// Timing sind die Zwischenzeiten des Phasenautomaten. Sie sind fest: keine Strategie darf
// sie veraendern.
type Timing struct {
	Yellow    time.Duration
	AllRed    time.Duration
	RedYellow time.Duration
}

// Intergreen ist die Summe der Zwischenzeiten eines Wechsels.
func (t Timing) Intergreen() time.Duration { return t.Yellow + t.AllRed + t.RedYellow }

// Machine ist der Phasenautomat. Er kennt Signalbilder und Zwischenzeiten, aber keine
// Nachfrage: wann eine Freigabe endet, entscheidet die Strategie.
type Machine struct {
	timing Timing
	state  State
}

// NewMachine startet mit Allrot. Erst danach gibt die erste Phase frei.
func NewMachine(timing Timing, now time.Time) *Machine {
	return &Machine{
		timing: timing,
		state:  State{Phase: PhaseStartup, Stage: StageAllRed, Since: now},
	}
}

func (m *Machine) State() State { return m.state }

func (m *Machine) SetTarget(target time.Duration) { m.state.Target = target }

// Advance geht einen Abschnitt weiter, sobald dessen Zeit abgelaufen ist. endGreen ist der
// Wunsch der Strategie, die laufende Freigabe zu beenden. Der Rueckgabewert sagt, ob sich
// der Zustand geaendert hat.
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

// Hold setzt den Beginn des laufenden Abschnitts neu. Die Blinkquittung nutzt das, damit
// nach ihr die volle Allrotzeit gilt.
func (m *Machine) Hold(now time.Time) { m.state.Since = now }

// Fault setzt den Automaten in den Notzustand. Zurueck fuehrt nur ein Neustart.
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
