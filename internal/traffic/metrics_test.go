package traffic

import (
	"testing"
	"time"

	"ampel/internal/light"
)

func TestMetricsMeanPerDirectionAndTotal(t *testing.T) {
	var m Metrics
	m.Add(Departure{Direction: light.North, Wait: 4 * time.Second})
	m.Add(Departure{Direction: light.North, Wait: 6 * time.Second})
	m.Add(Departure{Direction: light.East, Wait: 20 * time.Second})

	if got, want := m.Mean(light.North), 5*time.Second; got != want {
		t.Errorf("Mittel Nord %s, erwartet %s", got, want)
	}
	if got, want := m.Worst(light.North), 6*time.Second; got != want {
		t.Errorf("Maximum Nord %s, erwartet %s", got, want)
	}
	if got, want := m.Count(light.North), 2; got != want {
		t.Errorf("Anzahl Nord %d, erwartet %d", got, want)
	}
	if got, want := m.MeanAll(), 10*time.Second; got != want {
		t.Errorf("Mittel gesamt %s, erwartet %s", got, want)
	}
	if got, want := m.Total(), 3; got != want {
		t.Errorf("Anzahl gesamt %d, erwartet %d", got, want)
	}
}

func TestMetricsEmptyAndReset(t *testing.T) {
	var m Metrics
	if m.MeanAll() != 0 || m.Mean(light.West) != 0 {
		t.Error("leere Kennzahlen liefern nicht null")
	}
	m.Add(Departure{Direction: light.West, Wait: time.Second})
	m.Reset()
	if m.Total() != 0 || m.Mean(light.West) != 0 {
		t.Error("Reset liess Kennzahlen stehen")
	}
}
