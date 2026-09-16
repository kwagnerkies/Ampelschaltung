package controller

import (
	"time"

	"ampel/internal/light"
)

// Phase ist eine Freigabephase der Kreuzung. Nord und Sued sind gemeinsam gruen, danach Ost
// und West. PhaseStartup ist der Nullwert, damit ein frischer Automat nicht freigibt.
type Phase int

const (
	PhaseStartup Phase = iota
	PhaseNS
	PhaseEW
	PhaseFault
)

// Stage ist der Abschnitt innerhalb einer Phase. Die deutsche Signalfolge braucht neben der
// Freigabe drei Zwischenabschnitte.
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

// Directions sind die Zufahrten, die diese Phase freigibt.
func (p Phase) Directions() []light.Direction {
	switch p {
	case PhaseNS:
		return []light.Direction{light.North, light.South}
	case PhaseEW:
		return []light.Direction{light.East, light.West}
	}
	return nil
}

// PhaseOf ist die Freigabephase, zu der eine Zufahrt gehoert.
func PhaseOf(direction light.Direction) Phase {
	switch direction {
	case light.North, light.South:
		return PhaseNS
	}
	return PhaseEW
}

// Other ist die jeweils andere Freigabephase. Aus dem Start und aus der Stoerung heraus
// beginnt Nord und Sued.
func (p Phase) Other() Phase {
	switch p {
	case PhaseNS:
		return PhaseEW
	case PhaseEW:
		return PhaseNS
	}
	return PhaseNS
}

// State ist der Zustand des Phasenautomaten.
type State struct {
	Phase  Phase
	Stage  Stage
	Since  time.Time
	Target time.Duration
}

// Name ist die Bezeichnung des Zustands fuer das Log.
func (s State) Name() string {
	if s.Stage == StageAllRed {
		return "Allrot"
	}
	return s.Phase.String() + "_" + s.Stage.String()
}

// Aspects ist das Signalbild dieses Zustands. Alles, was nicht freigegeben ist, zeigt Rot.
func (s State) Aspects() [light.DirectionCount]light.Aspect {
	aspects := [light.DirectionCount]light.Aspect{
		light.AspectRed, light.AspectRed, light.AspectRed, light.AspectRed,
	}
	var aspect light.Aspect
	switch s.Stage {
	case StageGreen:
		aspect = light.AspectGreen
	case StageYellow:
		aspect = light.AspectYellow
	case StageRedYellow:
		aspect = light.AspectRedYellow
	default:
		return aspects
	}
	for _, direction := range s.Phase.Directions() {
		aspects[direction] = aspect
	}
	return aspects
}
