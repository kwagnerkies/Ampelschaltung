package strategy

import (
	"testing"
	"time"
)

var params = Params{
	MinGreen:   5 * time.Second,
	MaxGreen:   25 * time.Second,
	Cycle:      40 * time.Second,
	Intergreen: 6 * time.Second,
	Gap:        2 * time.Second,
	Extension:  1500 * time.Millisecond,
	MaxWait:    60 * time.Second,
}

func newAdaptive(t *testing.T) *Adaptive {
	t.Helper()
	a, err := NewAdaptive(params)
	if err != nil {
		t.Fatalf("NewAdaptive: %v", err)
	}
	return a
}

func TestCycleEffective(t *testing.T) {
	if got, want := params.CycleEffective(), 28*time.Second; got != want {
		t.Errorf("verteilbare Umlaufzeit %s, erwartet %s", got, want)
	}
}

// Asymmetrische Last muss asymmetrische Gruenzeiten ergeben. Das ist der Kern der Adaption.
func TestTargetGreenFollowsDemand(t *testing.T) {
	a := newAdaptive(t)
	cases := []struct {
		own, other float64
		want       time.Duration
	}{
		{1, 1, 14 * time.Second},
		{3, 1, 21 * time.Second},
		{1, 3, 7 * time.Second},
		{6, 1, 24 * time.Second},
		{0, 0, params.MinGreen},
	}
	for _, tc := range cases {
		got := a.TargetGreen(View{OwnDemand: tc.own, OtherDemand: tc.other})
		if delta := got - tc.want; delta < -100*time.Millisecond || delta > 100*time.Millisecond {
			t.Errorf("bei Nachfrage %v gegen %v ergibt sich %s, erwartet %s", tc.own, tc.other, got, tc.want)
		}
	}
}

// Die Zielzeit bleibt in den Grenzen, auch wenn die Nachfrage extrem einseitig ist.
func TestTargetGreenRespectsLimits(t *testing.T) {
	a := newAdaptive(t)
	if got := a.TargetGreen(View{OwnDemand: 100, OtherDemand: 0.01}); got != params.MaxGreen {
		t.Errorf("Zielzeit %s, erwartet die Hoechstgruenzeit %s", got, params.MaxGreen)
	}
	if got := a.TargetGreen(View{OwnDemand: 0.01, OtherDemand: 100}); got != params.MinGreen {
		t.Errorf("Zielzeit %s, erwartet die Mindestgruenzeit %s", got, params.MinGreen)
	}
}

func view(green time.Duration, target time.Duration) View {
	return View{
		Now:        start.Add(green),
		GreenSince: start,
		Target:     target,
	}
}

func TestEndGreenHoldsMinimumGreen(t *testing.T) {
	a := newAdaptive(t)
	v := view(time.Second, 10*time.Second)
	v.OtherQueue = 5
	v.OtherOldestWait = 5 * time.Minute
	v.LastStopLine = time.Minute

	if a.EndGreen(v) {
		t.Error("die Freigabe endete unter der Mindestgruenzeit")
	}
}

func TestEndGreenAtMaximum(t *testing.T) {
	a := newAdaptive(t)
	v := view(params.MaxGreen, params.MaxGreen)
	v.OwnQueue = 10
	v.LastStopLine = 0

	if !a.EndGreen(v) {
		t.Error("die Freigabe lief ueber die Hoechstgruenzeit hinaus")
	}
}

// Der Verhungerungsschutz greift, sobald die andere Seite zu lange wartet, unabhaengig von
// Zielzeit und laufender Verlaengerung.
func TestEndGreenStarvationProtection(t *testing.T) {
	a := newAdaptive(t)
	v := view(6*time.Second, 25*time.Second)
	v.OwnQueue = 6
	v.LastStopLine = 0
	v.OtherQueue = 1
	v.OtherOldestWait = params.MaxWait

	if !a.EndGreen(v) {
		t.Error("der Verhungerungsschutz griff nicht")
	}
}

// Ohne Anforderung der anderen Seite bleibt die Freigabe stehen, auch nach der Zielzeit.
func TestEndGreenHoldsWithoutDemand(t *testing.T) {
	a := newAdaptive(t)
	v := view(24*time.Second, 10*time.Second)
	v.OwnQueue = 2
	v.LastStopLine = time.Minute

	if a.EndGreen(v) {
		t.Error("die Freigabe endete ohne Anforderung der anderen Seite")
	}
}

func TestEndGreenIdleSwitchesEarly(t *testing.T) {
	a := newAdaptive(t)
	v := view(params.MinGreen, 20*time.Second)
	v.OwnQueue = 0
	v.OtherQueue = 3
	v.LastStopLine = 0

	if !a.EndGreen(v) {
		t.Error("die leere Freigabe lief weiter, obwohl die andere Seite wartet")
	}
}

func TestEndGreenGapAborts(t *testing.T) {
	a := newAdaptive(t)
	v := view(7*time.Second, 20*time.Second)
	v.OwnQueue = 2
	v.OtherQueue = 2
	v.LastStopLine = params.Gap

	if !a.EndGreen(v) {
		t.Error("die Freigabe lief trotz Luecke weiter")
	}
}

// Jede Bewegung innerhalb der Lueckenzeit verlaengert die Freigabe, bis die
// Hoechstgruenzeit erreicht ist. Danach ist Schluss.
func TestExtensionStopsAtMaximum(t *testing.T) {
	a := newAdaptive(t)
	target := 6 * time.Second
	green := time.Duration(0)
	step := 500 * time.Millisecond
	movement := time.Duration(0)

	for green < 40*time.Second {
		v := view(green, target)
		v.OwnQueue = 4
		v.OtherQueue = 4
		v.LastStopLine = green - movement
		if green-movement >= time.Second {
			// Ein Fahrzeug alle Sekunde: die Luecke wird nie erreicht.
			movement = green
			v.LastStopLine = 0
		}
		if a.EndGreen(v) {
			break
		}
		green += step
	}

	if green < params.MaxGreen {
		t.Errorf("die Verlaengerung endete bereits nach %s", green)
	}
	if green > params.MaxGreen+step {
		t.Errorf("die Freigabe lief %s, erwartet hoechstens %s", green, params.MaxGreen)
	}
}

// Ohne Verlaengerung endet die Freigabe an der Zielzeit, sobald die andere Seite wartet.
func TestEndGreenAtTargetWithoutMovement(t *testing.T) {
	a := newAdaptive(t)
	target := 12 * time.Second
	v := view(target, target)
	v.OwnQueue = 3
	v.OtherQueue = 3
	v.LastStopLine = time.Second

	if !a.EndGreen(v) {
		t.Error("die Freigabe lief ueber die Zielzeit hinaus, obwohl nichts verlaengert wurde")
	}
}

func TestNewAdaptiveRejectsBadParams(t *testing.T) {
	broken := params
	broken.MaxGreen = time.Second
	if _, err := NewAdaptive(broken); err == nil {
		t.Error("unsinnige Grenzwerte wurden angenommen")
	}
}
