package learning

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Load liest das Histogramm. Fehlt die Datei, ist sie unlesbar oder passt ihre Version
// nicht, kommt ein leeres Histogramm zurueck. Der zweite Rueckgabewert nennt dann den Grund;
// er gehoert ins Log, ist aber kein Startfehler.
func Load(path string) (*Histogram, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return New(), fmt.Errorf("lernzustand %s nicht gelesen: %w", path, err)
	}
	var histogram Histogram
	if err := json.Unmarshal(data, &histogram); err != nil {
		return New(), fmt.Errorf("lernzustand %s ist unbrauchbar: %w", path, err)
	}
	if histogram.Version != Version {
		return New(), fmt.Errorf("lernzustand %s hat version %d, erwartet %d",
			path, histogram.Version, Version)
	}
	return &histogram, nil
}

// Save schreibt das Histogramm atomar: erst in eine temporaere Datei im Zielverzeichnis,
// dann umbenennen. Ein Absturz mitten im Schreiben hinterlaesst damit keinen halben Stand.
func Save(path string, histogram *Histogram) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("verzeichnis fuer %s anlegen: %w", path, err)
	}
	data, err := json.Marshal(histogram)
	if err != nil {
		return fmt.Errorf("lernzustand serialisieren: %w", err)
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".histogram-*")
	if err != nil {
		return fmt.Errorf("temporaere datei anlegen: %w", err)
	}
	name := temp.Name()
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		_ = os.Remove(name)
		return fmt.Errorf("lernzustand schreiben: %w", err)
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		_ = os.Remove(name)
		return fmt.Errorf("lernzustand sichern: %w", err)
	}
	if err := temp.Close(); err != nil {
		_ = os.Remove(name)
		return fmt.Errorf("temporaere datei schliessen: %w", err)
	}
	if err := os.Rename(name, path); err != nil {
		_ = os.Remove(name)
		return fmt.Errorf("lernzustand nach %s umbenennen: %w", path, err)
	}
	return nil
}
