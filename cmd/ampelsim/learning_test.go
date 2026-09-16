package main

import (
	"testing"
	"time"

	"ampel/internal/config"
	"ampel/internal/controller"
	"ampel/internal/learning"
	"ampel/internal/light"
)

// trainingWindow ist der simulierte Abschnitt je Tag: die Stunden vor und in der
// Morgenspitze. Mehr Tageszeit muss nicht simuliert werden, um das Muster zu lernen.
const (
	trainingStartHour = 6
	trainingWindow    = 3 * time.Hour
	measureWindow     = 5 * time.Minute
)

func dayAt(day, hour int) time.Time {
	return time.Date(2026, 6, day, hour, 0, 0, 0, time.UTC)
}

// Nach mehreren simulierten Tagen mit demselben Muster muss die Zielgruenzeit vor der
// Lastspitze steigen, und der Reset muss dieses Verhalten wieder entfernen.
func TestLearningRaisesGreenBeforePeak(t *testing.T) {
	cfg := config.Default()
	// Gleiche Grundrate auf allen Zufahrten: die Asymmetrie entsteht allein aus dem
	// Tagesgang, Nord und Sued haben ihre Spitze um acht Uhr.
	rates := [light.DirectionCount]float64{0.06, 0.06, 0.06, 0.06}
	histogram := learning.New()

	for day := 1; day <= 5; day++ {
		if _, err := simulate(simOptions{
			config:    &cfg,
			mode:      "adaptiv",
			seed:      int64(day),
			rates:     rates,
			amplitude: 0.8,
			start:     dayAt(day, trainingStartHour),
			histogram: histogram,
		}, trainingWindow, false); err != nil {
			t.Fatalf("tag %d: %v", day, err)
		}
	}
	if histogram.Samples() == 0 {
		t.Fatal("das Histogramm blieb leer")
	}

	trained := measureTarget(t, &cfg, rates, histogram)
	fresh := measureTarget(t, &cfg, rates, learning.New())
	if trained <= fresh {
		t.Errorf("gelernt %s, ungelernt %s; erwartet wurde mehr Gruen vor der Spitze", trained, fresh)
	}

	// Der Reset loescht das Gelernte, damit die Vorfuehrung wieder von vorn beginnt.
	histogram.Reset()
	if histogram.Samples() != 0 {
		t.Fatalf("%d Beobachtungen nach dem Reset", histogram.Samples())
	}
	afterReset := measureTarget(t, &cfg, rates, histogram)
	if afterReset != fresh {
		t.Errorf("nach dem Reset %s, ungelernt %s; erwartet wurde derselbe Wert", afterReset, fresh)
	}
}

// measureTarget ist die mittlere Zielgruenzeit fuer Nord und Sued in den ersten Minuten vor
// der Morgenspitze.
func measureTarget(t *testing.T, cfg *config.Config, rates [light.DirectionCount]float64, histogram *learning.Histogram) time.Duration {
	t.Helper()
	s, err := newSimulation(simOptions{
		config:    cfg,
		mode:      "adaptiv",
		seed:      99,
		rates:     rates,
		amplitude: 0.8,
		start:     dayAt(6, 7),
		histogram: histogram,
	})
	if err != nil {
		t.Fatalf("newSimulation: %v", err)
	}
	s.walk(measureWindow, nil)
	if err := s.close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	return s.meanTarget(controller.PhaseNS, measureWindow)
}

// Der Lernzustand muss einen Neustart ueberleben.
func TestLearningSurvivesRestart(t *testing.T) {
	cfg := config.Default()
	path := t.TempDir() + "/histogram.json"
	rates := [light.DirectionCount]float64{0.06, 0.06, 0.06, 0.06}

	if _, err := simulate(simOptions{
		config:    &cfg,
		mode:      "adaptiv",
		seed:      1,
		rates:     rates,
		amplitude: 0.8,
		start:     dayAt(1, trainingStartHour),
		histogram: learning.New(),
		learnPath: path,
	}, time.Hour, false); err != nil {
		t.Fatalf("simulate: %v", err)
	}

	loaded, err := learning.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Samples() == 0 {
		t.Error("der gespeicherte Lernzustand ist leer")
	}
}
