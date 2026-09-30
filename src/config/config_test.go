package config

import (
	"errors"
	"io/fs"
	"path/filepath"
	"reflect"
	"strings"
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

func TestParseRejectsInvalid(t *testing.T) {
	cases := []struct {
		name string
		yaml string
		want string
	}{
		{
			name: "pin doppelt belegt",
			yaml: "hardware:\n  power_switch: 23\n",
			want: "BCM 23 ist doppelt belegt",
		},
		{
			name: "pin ausserhalb der Stiftleiste",
			yaml: "hardware:\n  power_switch: 40\n",
			want: "liegt ausserhalb von 0 bis 27",
		},
		{
			name: "grundgruen ueber hoechstgruen",
			yaml: "timing:\n  base_green_ms: 40000\n",
			want: "darf timing.max_green_ms",
		},
		{
			name: "gelbzeit null",
			yaml: "timing:\n  yellow_ms: 0\n",
			want: "timing.yellow_ms muss groesser als null sein",
		},
		{
			name: "unbekanntes feld",
			yaml: "timing:\n  gruen_ms: 5000\n",
			want: "field gruen_ms not found",
		},
		{
			name: "dauer als text",
			yaml: "timing:\n  follow_ms: zwei sekunden\n",
			want: "ganzzahlige millisekunden erwartet",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse([]byte(tc.yaml), "test")
			if err == nil {
				t.Fatalf("Konfiguration wurde angenommen, erwartet wurde ein Fehler mit %q", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("Fehler %q enthaelt nicht %q", err, tc.want)
			}
		})
	}
}

func TestValidateCollectsAllErrors(t *testing.T) {
	cfg := Default()
	cfg.Hardware.Chip = ""
	cfg.Hardware.Lamps.Brightness = 0
	err := cfg.Validate()
	if err == nil {
		t.Fatal("zwei Verstoesse wurden nicht gemeldet")
	}
	for _, want := range []string{"hardware.chip", "hardware.lamps.brightness"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Fehler %q enthaelt nicht %q", err, want)
		}
	}
}
