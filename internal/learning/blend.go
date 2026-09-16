package learning

import (
	"fmt"
	"time"

	"ampel/internal/light"
)

// Blender mischt die gemessene Nachfrage mit der Prognose des Histogramms. Das Gewicht der
// Prognose waechst mit der Zahl der Beobachtungen, deshalb reagiert das System frisch nach
// dem Reset rein reaktiv und spaeter vorausschauend.
type Blender struct {
	histogram *Histogram
	alpha     float64
	k         float64

	saveEvery time.Duration
	lastSave  time.Time
	save      func(*Histogram) error
	onError   func(error)
}

// Options beschreibt die Lernkomponente. save und onError duerfen fehlen; dann wird nichts
// gespeichert.
type Options struct {
	Histogram *Histogram
	Alpha     float64
	K         float64
	SaveEvery time.Duration
	Save      func(*Histogram) error
	OnError   func(error)
	Now       time.Time
}

func NewBlender(options Options) (*Blender, error) {
	if options.Histogram == nil {
		return nil, fmt.Errorf("kein histogramm uebergeben")
	}
	if options.Alpha <= 0 || options.Alpha > 1 {
		return nil, fmt.Errorf("lernrate %v liegt nicht zwischen null und eins", options.Alpha)
	}
	if options.K <= 0 {
		return nil, fmt.Errorf("mischkonstante %v ist nicht positiv", options.K)
	}
	return &Blender{
		histogram: options.Histogram,
		alpha:     options.Alpha,
		k:         options.K,
		saveEvery: options.SaveEvery,
		lastSave:  options.Now,
		save:      options.Save,
		onError:   options.OnError,
	}, nil
}

// Blend mischt Messung und Prognose und liefert zusaetzlich das Gewicht der Prognose.
func (b *Blender) Blend(at time.Time, direction light.Direction, live float64) (float64, float64) {
	predicted, samples := b.histogram.Predict(at, direction)
	weight := float64(samples) / (float64(samples) + b.k)
	return (1-weight)*live + weight*predicted, weight
}

// Observe uebernimmt die gemessene Nachfrage in das Zeitfenster, in dem die Phase begann.
func (b *Blender) Observe(at time.Time, direction light.Direction, demand float64) {
	b.histogram.Observe(at, direction, demand, b.alpha)
}

// Persist schreibt den Lernzustand, wenn das Speicherintervall abgelaufen ist. Der Aufruf
// erfolgt im Takt des Regelkreises, damit kein zweiter Goroutine das Histogramm anfasst.
func (b *Blender) Persist(at time.Time) {
	if b.save == nil || b.saveEvery <= 0 || at.Sub(b.lastSave) < b.saveEvery {
		return
	}
	b.lastSave = at
	b.Flush()
}

// Flush schreibt den Lernzustand sofort. Fuer das geordnete Beenden.
func (b *Blender) Flush() {
	if b.save == nil {
		return
	}
	if err := b.save(b.histogram); err != nil && b.onError != nil {
		b.onError(err)
	}
}

func (b *Blender) Reset() { b.histogram.Reset() }

// Histogram gibt den Lernzustand heraus, etwa zum Speichern beim Beenden.
func (b *Blender) Histogram() *Histogram { return b.histogram }
