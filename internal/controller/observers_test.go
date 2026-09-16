package controller

import (
	"errors"
	"testing"
	"time"
)

type counter struct {
	NopObserver
	calls int
}

func (c *counter) PhaseChanged(time.Time, State, string) { c.calls++ }
func (c *counter) Reset(time.Time)                       { c.calls++ }
func (c *counter) Fault(time.Time, error)                { c.calls++ }
func (c *counter) Sample(time.Time, Snapshot)            { c.calls++ }

// Jeder Beobachter muss jedes Ereignis sehen, sonst fehlt der Anzeige oder dem Log etwas.
func TestObserversReachEveryone(t *testing.T) {
	first, second := &counter{}, &counter{}
	list := Observers{first, second}

	list.PhaseChanged(time.Time{}, State{}, "adaptiv")
	list.Reset(time.Time{})
	list.Fault(time.Time{}, errors.New("test"))
	list.Sample(time.Time{}, Snapshot{})

	for i, observer := range []*counter{first, second} {
		if observer.calls != 4 {
			t.Errorf("Beobachter %d sah %d von vier Ereignissen", i, observer.calls)
		}
	}
}

func TestEmptyObserversAreHarmless(t *testing.T) {
	var list Observers
	list.PhaseChanged(time.Time{}, State{}, "festzeit")
	list.Sample(time.Time{}, Snapshot{})
}
