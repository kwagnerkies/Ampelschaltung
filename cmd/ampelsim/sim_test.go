package main

import (
	"testing"
	"time"

	"ampel/internal/config"
	"ampel/internal/light"
)

func simulate(t *testing.T, options simOptions, duration time.Duration) result {
	t.Helper()
	s, err := newSimulation(options)
	if err != nil {
		t.Fatalf("newSimulation: %v", err)
	}
	s.walk(duration, nil)
	if err := s.close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	return s.result()
}

func options(rates [light.DirectionCount]float64) simOptions {
	cfg := config.Default()
	return simOptions{
		config: &cfg,
		seed:   7,
		rates:  rates,
		start:  time.Date(2026, 1, 5, 7, 0, 0, 0, time.UTC),
	}
}

// Dichter Verkehr auf einer Achse muss abfliessen, und keine Richtung darf stehenbleiben.
func TestTrafficFlowsOnBothAxes(t *testing.T) {
	r := simulate(t, options([light.DirectionCount]float64{0.10, 0.02, 0.10, 0.02}), 20*time.Minute)

	if r.arrived == 0 {
		t.Fatal("kein Fahrzeug erzeugt")
	}
	if r.departed < r.arrived*9/10 {
		t.Errorf("%d von %d Fahrzeugen abgefahren", r.departed, r.arrived)
	}
	if r.worst > 90*time.Second {
		t.Errorf("laengste Wartezeit %s, das sieht nach einer verhungerten Richtung aus", r.worst)
	}
}

// Ohne Verkehr laeuft die Kreuzung weiter, ohne dass etwas haengt.
func TestSimulationWithoutTraffic(t *testing.T) {
	r := simulate(t, options([light.DirectionCount]float64{}), 5*time.Minute)
	if r.arrived != 0 || r.departed != 0 {
		t.Errorf("%d Ankuenfte und %d Abfahrten ohne Verkehr", r.arrived, r.departed)
	}
}
