package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// Load liest die Konfiguration von path, ergaenzt fehlende Felder mit den Defaults und
// validiert das Ergebnis. Fehlt die Datei, ist der Fehler mit fs.ErrNotExist vergleichbar.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("konfiguration lesen: %w", err)
	}
	return Parse(data, path)
}

// Parse verarbeitet den Inhalt einer Konfigurationsdatei. name dient nur der Fehlermeldung.
func Parse(data []byte, name string) (*Config, error) {
	cfg := Default()
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("konfiguration %s: %w", name, err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("konfiguration %s ungueltig:\n%w", name, err)
	}
	return &cfg, nil
}
