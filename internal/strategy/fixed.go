package strategy

import "time"

// Fixed ist die Festzeitsteuerung: feste Gruenzeit, keine Verlaengerung, keine Auswertung
// der Nachfrage. Die Sensoren laufen weiter, denn die Wartezeiten werden auch hier gemessen.
type Fixed struct {
	green time.Duration
}

var _ Strategy = (*Fixed)(nil)

func NewFixed(green time.Duration) *Fixed { return &Fixed{green: green} }

func (f *Fixed) Name() string { return "festzeit" }

func (f *Fixed) TargetGreen(View) time.Duration { return f.green }

func (f *Fixed) EndGreen(view View) bool { return view.Green() >= f.green }
