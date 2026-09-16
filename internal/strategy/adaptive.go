package strategy

import "time"

// eps verhindert die Division durch null, wenn keine Zufahrt Nachfrage meldet.
const eps = 1e-9

// Adaptive verteilt die Umlaufzeit nach der geglaetteten Nachfrage und arbeitet innerhalb
// der Freigabe verkehrsabhaengig: Bewegung an der Haltelinie verlaengert, eine Luecke bricht
// ab, und der Verhungerungsschutz erzwingt den Wechsel.
type Adaptive struct {
	params Params

	// greenSince erkennt den Beginn einer neuen Freigabe.
	greenSince time.Time
	// limit ist die Zielzeit zuzueglich aller Verlaengerungen dieser Freigabe.
	limit time.Duration
	// lastGap ist der zuletzt gesehene Abstand zur Haltelinienbewegung. Sinkt er, war eine
	// neue Flanke dabei.
	lastGap time.Duration
}

var _ Strategy = (*Adaptive)(nil)

func NewAdaptive(params Params) (*Adaptive, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return &Adaptive{params: params}, nil
}

func (a *Adaptive) Name() string { return "adaptiv" }

// TargetGreen verteilt die verteilbare Umlaufzeit im Verhaeltnis der Nachfragen.
func (a *Adaptive) TargetGreen(view View) time.Duration {
	share := view.OwnDemand / (view.OwnDemand + view.OtherDemand + eps)
	target := time.Duration(float64(a.params.CycleEffective()) * share)
	return a.params.clamp(target)
}

// EndGreen entscheidet im Takt des Regelkreises. Der Aufruf fuehrt die Verlaengerung mit,
// deshalb ist er nicht seitenwirkungsfrei.
func (a *Adaptive) EndGreen(view View) bool {
	a.track(view)
	green := view.Green()

	if green >= a.params.MaxGreen {
		return true
	}
	// Die Mindestgruenzeit ist die harte Untergrenze. Auch der Verhungerungsschutz
	// unterschreitet sie nicht, sonst entstehen Freigaben unter der Sicherheitsgrenze.
	if green < a.params.MinGreen {
		return false
	}
	// Verhungerungsschutz: er hat Vorrang vor Zielzeit, Luecke und Leerlauf.
	if view.OtherOldestWait >= a.params.MaxWait {
		return true
	}
	if !demanding(view) {
		// Ohne Anforderung der anderen Seite bleibt die Freigabe stehen. Das ist das
		// Verhalten ohne Verkehr.
		return false
	}
	if view.OwnQueue == 0 {
		return true
	}
	if view.LastStopLine >= a.params.Gap {
		return true
	}
	return green >= a.limit
}

// track uebernimmt die Zielzeit einer neuen Freigabe und verlaengert bei jeder Bewegung an
// der Haltelinie um extension, hoechstens bis zur Hoechstgruenzeit.
func (a *Adaptive) track(view View) {
	if a.greenSince != view.GreenSince {
		a.greenSince = view.GreenSince
		a.limit = a.params.clamp(view.Target)
		a.lastGap = view.LastStopLine
		return
	}
	if view.LastStopLine < a.lastGap && view.LastStopLine < a.params.Gap {
		a.limit += a.params.Extension
		if a.limit > a.params.MaxGreen {
			a.limit = a.params.MaxGreen
		}
	}
	a.lastGap = view.LastStopLine
}

// demanding sagt, ob die andere Seite ueberhaupt freigegeben werden will.
func demanding(view View) bool {
	return view.OtherQueue > 0 || view.OtherOldestWait > 0
}
