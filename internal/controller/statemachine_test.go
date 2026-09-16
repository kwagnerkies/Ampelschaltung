package controller

import (
	"testing"
	"time"

	"ampel/internal/light"
)

var timing = Timing{
	Yellow:    3 * time.Second,
	AllRed:    2 * time.Second,
	RedYellow: time.Second,
}

var start = time.Date(2026, 4, 1, 7, 0, 0, 0, time.UTC)

// Der Automat muss die Folge Gruen, Gelb, Allrot, RotGelb, Gruen der anderen Phase genau
// einhalten. Der Test prueft jeden Abschnitt einschliesslich seiner Dauer.
func TestMachineRunsCompleteCycle(t *testing.T) {
	m := NewMachine(timing, start)
	now := start

	steps := []struct {
		wait     time.Duration
		endGreen bool
		phase    Phase
		stage    Stage
	}{
		{timing.AllRed, false, PhaseNS, StageRedYellow},
		{timing.RedYellow, false, PhaseNS, StageGreen},
		{5 * time.Second, true, PhaseNS, StageYellow},
		{timing.Yellow, false, PhaseNS, StageAllRed},
		{timing.AllRed, false, PhaseEW, StageRedYellow},
		{timing.RedYellow, false, PhaseEW, StageGreen},
		{5 * time.Second, true, PhaseEW, StageYellow},
		{timing.Yellow, false, PhaseEW, StageAllRed},
		{timing.AllRed, false, PhaseNS, StageRedYellow},
	}

	if got := m.State(); got.Phase != PhaseStartup || got.Stage != StageAllRed {
		t.Fatalf("Startzustand %s, erwartet Start und Allrot", got.Name())
	}

	for i, step := range steps {
		before := now
		// Ohne Wunsch der Strategie darf kein Abschnitt vorzeitig enden.
		now = now.Add(step.wait - time.Millisecond)
		if m.Advance(now, false) {
			t.Fatalf("schritt %d: Wechsel schon nach %s", i, now.Sub(before))
		}
		now = now.Add(time.Millisecond)
		if !m.Advance(now, step.endGreen) {
			t.Fatalf("schritt %d: kein Wechsel nach %s", i, now.Sub(before))
		}
		if got := m.State(); got.Phase != step.phase || got.Stage != step.stage {
			t.Fatalf("schritt %d ergibt %s, erwartet %s_%s", i, got.Name(), step.phase, step.stage)
		}
	}
}

// Ohne Wunsch der Strategie bleibt Gruen stehen. Das ist das Verhalten ohne Verkehr.
func TestMachineHoldsGreenWithoutStrategy(t *testing.T) {
	m := NewMachine(timing, start)
	now := start.Add(timing.AllRed)
	m.Advance(now, false)
	now = now.Add(timing.RedYellow)
	m.Advance(now, false)
	if m.State().Stage != StageGreen {
		t.Fatalf("Zustand %s, erwartet Gruen", m.State().Name())
	}

	for i := 0; i < 100; i++ {
		now = now.Add(time.Second)
		if m.Advance(now, false) {
			t.Fatalf("Gruen endete nach %s ohne Anforderung", now.Sub(start))
		}
	}
}

func TestStateAspects(t *testing.T) {
	cases := []struct {
		state State
		want  [light.DirectionCount]light.Aspect
	}{
		{State{Phase: PhaseNS, Stage: StageGreen},
			[light.DirectionCount]light.Aspect{light.AspectGreen, light.AspectRed, light.AspectGreen, light.AspectRed}},
		{State{Phase: PhaseNS, Stage: StageYellow},
			[light.DirectionCount]light.Aspect{light.AspectYellow, light.AspectRed, light.AspectYellow, light.AspectRed}},
		{State{Phase: PhaseEW, Stage: StageRedYellow},
			[light.DirectionCount]light.Aspect{light.AspectRed, light.AspectRedYellow, light.AspectRed, light.AspectRedYellow}},
		{State{Phase: PhaseEW, Stage: StageAllRed},
			[light.DirectionCount]light.Aspect{light.AspectRed, light.AspectRed, light.AspectRed, light.AspectRed}},
		{State{Phase: PhaseStartup, Stage: StageAllRed},
			[light.DirectionCount]light.Aspect{light.AspectRed, light.AspectRed, light.AspectRed, light.AspectRed}},
	}
	for _, tc := range cases {
		if got := tc.state.Aspects(); got != tc.want {
			t.Errorf("%s ergibt %v, erwartet %v", tc.state.Name(), got, tc.want)
		}
	}
	for _, tc := range cases {
		if err := Check(tc.state.Aspects()); err != nil {
			t.Errorf("%s ist unzulaessig: %v", tc.state.Name(), err)
		}
	}
}

func TestTargetClearedOutsideGreen(t *testing.T) {
	m := NewMachine(timing, start)
	now := start.Add(timing.AllRed)
	m.Advance(now, false)
	now = now.Add(timing.RedYellow)
	m.Advance(now, false)
	m.SetTarget(12 * time.Second)

	now = now.Add(time.Second)
	m.Advance(now, true)
	if got := m.State().Target; got != 0 {
		t.Errorf("Zielgruenzeit %s in %s, erwartet null", got, m.State().Name())
	}
}

func TestFaultEntersFlashState(t *testing.T) {
	m := NewMachine(timing, start)
	m.Fault(start.Add(time.Minute))
	if got := m.State(); got.Phase != PhaseFault {
		t.Fatalf("Zustand %s, erwartet Stoerung", got.Name())
	}
}
