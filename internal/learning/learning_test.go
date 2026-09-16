package learning

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"ampel/internal/light"
)

var day = time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

func at(hour, minute int) time.Time {
	return day.Add(time.Duration(hour)*time.Hour + time.Duration(minute)*time.Minute)
}

func TestSlotIndex(t *testing.T) {
	cases := []struct {
		hour, minute int
		want         int
	}{
		{0, 0, 0},
		{0, 14, 0},
		{0, 15, 1},
		{8, 0, 32},
		{23, 59, 95},
	}
	for _, tc := range cases {
		if got := SlotIndex(at(tc.hour, tc.minute)); got != tc.want {
			t.Errorf("%02d:%02d liegt in Fenster %d, erwartet %d", tc.hour, tc.minute, got, tc.want)
		}
	}
}

// Bei wiederholtem Muster muss das Histogramm gegen den beobachteten Wert konvergieren.
func TestHistogramConverges(t *testing.T) {
	h := New()
	for day := 0; day < 40; day++ {
		h.Observe(at(8, 5), light.North, 6, 0.1)
	}
	predicted, samples := h.Predict(at(8, 10), light.North)

	if samples != 40 {
		t.Errorf("%d Beobachtungen, erwartet 40", samples)
	}
	if predicted < 5.9 || predicted > 6.1 {
		t.Errorf("Prognose %v, erwartet etwa 6", predicted)
	}
	if other, _ := h.Predict(at(17, 0), light.North); other != 0 {
		t.Errorf("das Abendfenster lernte %v mit", other)
	}
}

// Das Gewicht der Prognose waechst mit der Zahl der Beobachtungen: w = n / (n + k).
func TestBlendWeightGrowsWithSamples(t *testing.T) {
	h := New()
	b := newBlender(t, h)

	blended, weight := b.Blend(at(8, 0), light.North, 2)
	if weight != 0 || blended != 2 {
		t.Fatalf("frisch: Mischung %v mit Gewicht %v, erwartet die reine Messung", blended, weight)
	}

	for i := 0; i < 10; i++ {
		b.Observe(at(8, 0), light.North, 6)
	}
	blended, weight = b.Blend(at(8, 0), light.North, 2)
	if weight < 0.49 || weight > 0.51 {
		t.Errorf("Gewicht %v nach zehn Beobachtungen, erwartet 0.5", weight)
	}
	// Halbes Gewicht auf Messung 2 und Prognose 6 ergibt genau 4.
	if blended < 3.9 || blended > 4.1 {
		t.Errorf("Mischung %v, erwartet etwa 4", blended)
	}

	for i := 0; i < 90; i++ {
		b.Observe(at(8, 0), light.North, 6)
	}
	_, weight = b.Blend(at(8, 0), light.North, 2)
	if weight < 0.9 {
		t.Errorf("Gewicht %v nach hundert Beobachtungen, erwartet ueber 0.9", weight)
	}
}

func TestResetClearsHistogram(t *testing.T) {
	h := New()
	b := newBlender(t, h)
	for i := 0; i < 20; i++ {
		b.Observe(at(8, 0), light.North, 6)
	}

	b.Reset()

	if h.Samples() != 0 {
		t.Errorf("%d Beobachtungen nach dem Reset", h.Samples())
	}
	blended, weight := b.Blend(at(8, 0), light.North, 2)
	if weight != 0 || blended != 2 {
		t.Errorf("nach dem Reset: Mischung %v mit Gewicht %v, erwartet die reine Messung", blended, weight)
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "unter", "histogram.json")
	h := New()
	h.Observe(at(8, 0), light.South, 4, 0.1)

	if err := Save(path, h); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	predicted, samples := loaded.Predict(at(8, 0), light.South)
	if samples != 1 || predicted != 4 {
		t.Errorf("geladen: %v aus %d Beobachtungen", predicted, samples)
	}
}

// Eine defekte oder fehlende Datei fuehrt zu einem leeren Start, nicht zum Absturz.
func TestLoadFallsBackToEmpty(t *testing.T) {
	dir := t.TempDir()
	broken := filepath.Join(dir, "kaputt.json")
	if err := os.WriteFile(broken, []byte("{kein json"), 0o644); err != nil {
		t.Fatalf("schreiben: %v", err)
	}
	oldVersion := filepath.Join(dir, "alt.json")
	if err := os.WriteFile(oldVersion, []byte(`{"version":99}`), 0o644); err != nil {
		t.Fatalf("schreiben: %v", err)
	}

	for _, path := range []string{broken, oldVersion, filepath.Join(dir, "fehlt.json")} {
		h, err := Load(path)
		if err == nil {
			t.Errorf("%s: kein Hinweis auf den Grund", filepath.Base(path))
		}
		if h == nil || h.Samples() != 0 {
			t.Errorf("%s: kein leeres Histogramm", filepath.Base(path))
		}
	}
}

// Gespeichert wird nur, wenn das Intervall abgelaufen ist.
func TestPersistHonoursInterval(t *testing.T) {
	h := New()
	saves := 0
	b, err := NewBlender(Options{
		Histogram: h,
		Alpha:     0.1,
		K:         10,
		SaveEvery: time.Minute,
		Save:      func(*Histogram) error { saves++; return nil },
		Now:       at(8, 0),
	})
	if err != nil {
		t.Fatalf("NewBlender: %v", err)
	}

	b.Persist(at(8, 0).Add(30 * time.Second))
	if saves != 0 {
		t.Errorf("%d Sicherungen vor Ablauf des Intervalls", saves)
	}
	b.Persist(at(8, 0).Add(61 * time.Second))
	if saves != 1 {
		t.Errorf("%d Sicherungen nach Ablauf des Intervalls, erwartet eine", saves)
	}
	b.Flush()
	if saves != 2 {
		t.Errorf("%d Sicherungen nach Flush, erwartet zwei", saves)
	}
}

func TestNewBlenderRejectsBadOptions(t *testing.T) {
	if _, err := NewBlender(Options{Alpha: 0.1, K: 10}); err == nil {
		t.Error("fehlendes Histogramm wurde angenommen")
	}
	if _, err := NewBlender(Options{Histogram: New(), Alpha: 0, K: 10}); err == nil {
		t.Error("Lernrate null wurde angenommen")
	}
	if _, err := NewBlender(Options{Histogram: New(), Alpha: 0.1, K: 0}); err == nil {
		t.Error("Mischkonstante null wurde angenommen")
	}
}

func newBlender(t *testing.T, h *Histogram) *Blender {
	t.Helper()
	b, err := NewBlender(Options{Histogram: h, Alpha: 0.1, K: 10, Now: day})
	if err != nil {
		t.Fatalf("NewBlender: %v", err)
	}
	return b
}
