package config

import (
	"errors"
	"io/fs"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDefaultIsValid(t *testing.T) {
	cfg := Default()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Defaults muessen gueltig sein: %v", err)
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
