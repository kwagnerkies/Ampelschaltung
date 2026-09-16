package main

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ampel/internal/light"
)

func TestLaneQueueMovesAndReportsEdges(t *testing.T) {
	l := newLane(light.North, []int{5, 6, 13}, simStart)
	now := simStart

	// Ein Fahrzeug kommt an und rueckt an die Linie vor.
	l.arrive(now)
	inputs := l.step(now, false)
	if len(inputs) != 1 || inputs[0].Pin != 5 || !inputs[0].Active {
		t.Fatalf("Flanken %+v, erwartet Haltelinie belegt", inputs)
	}

	// Bei Rot bleibt es stehen.
	now = now.Add(5 * time.Second)
	if inputs := l.step(now, false); len(inputs) != 0 {
		t.Fatalf("Flanken bei Rot: %+v", inputs)
	}
	if l.departed != 0 {
		t.Fatal("das Fahrzeug fuhr bei Rot ab")
	}

	// Bei Gruen faehrt es nach der Ueberfahrtzeit ab.
	now = now.Add(crossingTime)
	inputs = l.step(now, true)
	if len(inputs) != 1 || inputs[0].Active {
		t.Fatalf("Flanken %+v, erwartet Haltelinie frei", inputs)
	}
	if l.departed != 1 {
		t.Fatalf("%d Abfahrten, erwartet eine", l.departed)
	}
	if want := 5*time.Second + crossingTime; l.waitSum != want {
		t.Errorf("Wartezeit %s, erwartet %s", l.waitSum, want)
	}
}

// Drei stehende Fahrzeuge belegen alle drei Sensoren, ein viertes bleibt unsichtbar.
func TestLaneSensorsShowQueue(t *testing.T) {
	l := newLane(light.South, []int{16, 20, 21}, simStart)
	now := simStart
	for i := 0; i < 4; i++ {
		l.arrive(now)
	}
	l.step(now, false)

	for index, want := range []bool{true, true, true} {
		if got := l.sensor(index); got != want {
			t.Errorf("Sensor %d ist %v, erwartet %v", index, got, want)
		}
	}
	if l.waiting() != 4 {
		t.Errorf("%d Fahrzeuge in der Schlange, erwartet vier", l.waiting())
	}
}

func TestArrivalsFollowRate(t *testing.T) {
	rates := [light.DirectionCount]float64{0.5, 0, 0, 0}
	a := newArrivals(1, rates, simStart)

	count := 0
	now := simStart
	for elapsed := time.Duration(0); elapsed < 10*time.Minute; elapsed += simStep {
		now = now.Add(simStep)
		count += len(a.due(now))
	}
	// Erwartet werden etwa 300 Ankuenfte in zehn Minuten.
	if count < 240 || count > 360 {
		t.Errorf("%d Ankuenfte in zehn Minuten, erwartet etwa 300", count)
	}
}

// readVehiclesForTest zaehlt die Fahrzeugzeilen im Logverzeichnis.
func readVehiclesForTest(dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}
	total := 0
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "vehicles") {
			continue
		}
		file, err := os.Open(filepath.Join(dir, entry.Name()))
		if err != nil {
			return 0, err
		}
		reader := csv.NewReader(file)
		reader.Comma = ';'
		records, err := reader.ReadAll()
		_ = file.Close()
		if err != nil {
			return 0, err
		}
		total += len(records) - 1
	}
	return total, nil
}
