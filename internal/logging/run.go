package logging

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Run ist ein Messlauf mit eigener Kennung und eigenen Dateien. Der Reset-Taster beginnt
// einen neuen Lauf, damit die Auswertung die Abschnitte trennen kann.
type Run struct {
	dir      string
	buffer   int
	label    string
	id       string
	vehicles *Writer
	state    *Writer
	events   *Writer
}

// NewRun legt die drei Dateien des Laufs an. Die Kennung ist der Startzeitpunkt, ergaenzt um
// label, damit mehrere Laeufe derselben Sekunde unterscheidbar bleiben.
func NewRun(dir string, started time.Time, buffer int, label string) (*Run, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("logverzeichnis %s anlegen: %w", dir, err)
	}
	r := &Run{dir: dir, buffer: buffer, label: label}
	if err := r.open(started); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *Run) ID() string { return r.id }

func (r *Run) Vehicles() *Writer { return r.vehicles }

func (r *Run) State() *Writer { return r.state }

func (r *Run) Events() *Writer { return r.events }

// Dropped ist die Anzahl verworfener Zeilen ueber alle Dateien.
func (r *Run) Dropped() uint64 {
	return r.vehicles.Dropped() + r.state.Dropped() + r.events.Dropped()
}

// Rotate schliesst die Dateien und beginnt einen Lauf mit neuer Kennung.
func (r *Run) Rotate(at time.Time) error {
	if err := r.Close(); err != nil {
		return err
	}
	return r.open(at)
}

func (r *Run) Close() error {
	var errs []error
	for _, w := range []*Writer{r.vehicles, r.state, r.events} {
		if w != nil {
			errs = append(errs, w.Close())
		}
	}
	return errors.Join(errs...)
}

func (r *Run) open(at time.Time) error {
	r.id = at.Format("20060102-150405")
	if r.label != "" {
		r.id += "-" + r.label
	}
	files := []struct {
		name   string
		header []string
		target **Writer
	}{
		{"vehicles", VehicleHeader, &r.vehicles},
		{"state", StateHeader, &r.state},
		{"events", EventHeader, &r.events},
	}
	for _, file := range files {
		path := filepath.Join(r.dir, fmt.Sprintf("%s-%s.csv", file.name, r.id))
		w, err := NewWriter(path, file.header, r.buffer)
		if err != nil {
			return err
		}
		*file.target = w
	}
	return nil
}
