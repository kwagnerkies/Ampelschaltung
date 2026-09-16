package main

import (
	"testing"
	"time"

	"ampel/internal/config"
	"ampel/internal/light"
)

var simStart = time.Date(2026, 1, 5, 7, 0, 0, 0, time.UTC)

// Der Vergleich muss reproduzierbar einen Wartezeitvorteil der adaptiven Steuerung zeigen.
// Beide Laeufe bekommen dasselbe Ankunftsmuster, weil sie denselben Startwert verwenden.
func TestAdaptiveBeatsFixedOverThirtyMinutes(t *testing.T) {
	cfg := config.Default()
	// Lastniveau unterhalb der Saettigung: die Warteschlange bleibt meist im Erfassungsbereich,
	// damit die gemessene Wartezeit die tatsaechliche abbildet.
	rates := [light.DirectionCount]float64{0.08, 0.03, 0.08, 0.03}

	for _, seed := range []int64{1, 7, 42} {
		results := map[string]result{}
		for _, mode := range []string{"festzeit", "adaptiv"} {
			r, err := simulate(simOptions{
				config: &cfg,
				mode:   mode,
				seed:   seed,
				rates:  rates,
				start:  simStart,
			}, 30*time.Minute, false)
			if err != nil {
				t.Fatalf("seed %d, modus %s: %v", seed, mode, err)
			}
			results[mode] = r
		}
		fixed, adaptive := results["festzeit"], results["adaptiv"]

		if fixed.arrived != adaptive.arrived {
			t.Fatalf("seed %d: %d gegen %d Ankuenfte, erwartet identisches Muster", seed, fixed.arrived, adaptive.arrived)
		}
		if adaptive.truth >= fixed.truth {
			t.Errorf("seed %d: adaptiv wartet %s, festzeit %s", seed, adaptive.truth, fixed.truth)
		}
		if adaptive.measured > fixed.measured {
			t.Errorf("seed %d: gemessen adaptiv %s, festzeit %s", seed, adaptive.measured, fixed.measured)
		}
		// Unterhalb der Saettigung kommen in beiden Betriebsarten praktisch alle Fahrzeuge
		// durch. Der Unterschied liegt in der Wartezeit, nicht im Durchsatz.
		for _, r := range []result{fixed, adaptive} {
			if r.departed < r.arrived-5 {
				t.Errorf("seed %d, %s: nur %d von %d Fahrzeugen kamen durch", seed, r.mode, r.departed, r.arrived)
			}
		}
	}
}

func TestSimulationWritesLogFiles(t *testing.T) {
	cfg := config.Default()
	dir := t.TempDir()

	_, err := simulate(simOptions{
		config: &cfg,
		mode:   "adaptiv",
		seed:   3,
		rates:  [light.DirectionCount]float64{0.1, 0.1, 0.1, 0.1},
		start:  simStart,
		logDir: dir,
	}, 2*time.Minute, false)
	if err != nil {
		t.Fatalf("simulate: %v", err)
	}

	rows, err := readVehiclesForTest(dir)
	if err != nil {
		t.Fatalf("Fahrzeugdaten lesen: %v", err)
	}
	if rows == 0 {
		t.Error("keine Fahrzeugzeilen geschrieben")
	}
}

// Ueber der Saettigung verliert die Messung ihre Aussagekraft: was hinter dem letzten Sensor
// steht, sieht die Steuerung nicht, und die gemessene Wartezeit faellt, obwohl die
// tatsaechliche steigt. Der Test haelt diese Grenze der Auswertung fest.
func TestMeasurementUnderstatesWaitAboveSaturation(t *testing.T) {
	cfg := config.Default()
	rates := [light.DirectionCount]float64{0.25, 0.07, 0.25, 0.07}

	r, err := simulate(simOptions{
		config: &cfg,
		mode:   "festzeit",
		seed:   1,
		rates:  rates,
		start:  simStart,
	}, 30*time.Minute, false)
	if err != nil {
		t.Fatalf("simulate: %v", err)
	}
	if r.measured >= r.truth {
		t.Errorf("gemessen %s, tatsaechlich %s; erwartet wurde eine deutliche Untererfassung", r.measured, r.truth)
	}
	if r.truth < 3*r.measured {
		t.Errorf("die Untererfassung betraegt nur %s gegen %s", r.measured, r.truth)
	}
}

// Ohne Verkehr darf keine Stoerung auftreten und die Kreuzung muss weiterlaufen.
func TestSimulationWithoutTraffic(t *testing.T) {
	cfg := config.Default()
	r, err := simulate(simOptions{
		config: &cfg,
		mode:   "adaptiv",
		seed:   1,
		rates:  [light.DirectionCount]float64{},
		start:  simStart,
	}, 5*time.Minute, false)
	if err != nil {
		t.Fatalf("simulate: %v", err)
	}
	if r.arrived != 0 || r.departed != 0 {
		t.Errorf("ohne Verkehr: %d Ankuenfte, %d Abfahrten", r.arrived, r.departed)
	}
}
