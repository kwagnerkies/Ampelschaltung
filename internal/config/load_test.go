package config

import (
	"errors"
	"io/fs"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestDefaultIsValid(t *testing.T) {
	cfg := Default()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Defaults muessen gueltig sein: %v", err)
	}
}

func TestLoadShippedConfig(t *testing.T) {
	cfg, err := Load(filepath.Join("..", "..", "configs", "config.yaml"))
	if err != nil {
		t.Fatalf("configs/config.yaml laedt nicht: %v", err)
	}
	if got, want := cfg.Timing.Yellow.Duration(), 3*time.Second; got != want {
		t.Errorf("Gelbzeit %s, erwartet %s", got, want)
	}
	if got, want := cfg.Timing.Intergreen(), 6*time.Second; got != want {
		t.Errorf("Zwischenzeiten %s, erwartet %s", got, want)
	}
	if got, want := cfg.Timing.Intergreen(), 6*time.Second; got != want {
		t.Errorf("verteilbare Umlaufzeit %s, erwartet %s", got, want)
	}
	if got, want := cfg.Hardware.Sensors.North, []int{5, 6, 13}; !reflect.DeepEqual(got, want) {
		t.Errorf("Sensoren Nord %v, erwartet %v", got, want)
	}
	if got := cfg.Hardware.Sensors.SensorCount(); got != 3 {
		t.Errorf("Sensoranzahl %d, erwartet 3", got)
	}
}

func TestLoadMissingFileIsNotExist(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "fehlt.yaml"))
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Fehler %v, erwartet fs.ErrNotExist", err)
	}
}

func TestParseEmptyYieldsDefaults(t *testing.T) {
	cfg, err := Parse([]byte("# leer\n"), "test")
	if err != nil {
		t.Fatalf("leere Datei muss die Defaults ergeben: %v", err)
	}
	want := Default()
	if !reflect.DeepEqual(*cfg, want) {
		t.Errorf("Konfiguration %+v, erwartet %+v", *cfg, want)
	}
}

func TestParseOverridesSingleField(t *testing.T) {
	cfg, err := Parse([]byte("fixed:\n  green_ms: 12000\n"), "test")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got, want := cfg.Fixed.Green.Duration(), 12*time.Second; got != want {
		t.Errorf("Festzeitgruen %s, erwartet %s", got, want)
	}
	if got, want := cfg.Timing.MaxGreen, Default().Timing.MaxGreen; got != want {
		t.Errorf("max_green_ms %s, erwartet unveraenderten Default %s", got, want)
	}
}

// Die Rueckstautabelle wird ersetzt und nicht mit dem Default zusammengefuehrt, sonst
// bliebe bei zwei Sensoren der Eintrag fuer drei belegte Sensoren stehen.
