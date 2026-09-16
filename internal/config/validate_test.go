package config

import (
	"strings"
	"testing"
)

func TestParseRejectsInvalid(t *testing.T) {
	cases := []struct {
		name string
		yaml string
		want string
	}{
		{
			name: "pin doppelt belegt",
			yaml: "hardware:\n  mode_switch: 17\n",
			want: "BCM 17 ist doppelt belegt",
		},
		{
			name: "pin ausserhalb der Stiftleiste",
			yaml: "hardware:\n  reset_button: 40\n",
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
			name: "lampe fehlt in der bitreihenfolge",
			yaml: bitOrderYAML("N_red", "N_yellow", "free"),
			want: "fehlen die Lampen: N_green",
		},
		{
			name: "bitreihenfolge zu kurz",
			yaml: "hardware:\n  shift_register:\n    bit_order: [N_red, N_yellow, N_green]\n",
			want: "braucht genau 16 Eintraege",
		},
		{
			name: "unbekannter lampenname",
			yaml: bitOrderYAML("N_red", "N_yellow", "N_blau"),
			want: "kennt \"N_blau\" nicht",
		},
		{
			name: "ungleiche sensoranzahl",
			yaml: "hardware:\n  sensors:\n    west: [23, 24]\n",
			want: "hardware.sensors.west hat 2 Sensoren",
		},
		{
			name: "rueckstautabelle unvollstaendig",
			yaml: "queue_mapping:\n  0: 0\n  1: 1\n  2: 3\n",
			want: "fehlt der Eintrag fuer 3 belegte Sensoren",
		},
		{
			name: "rueckstautabelle nicht monoton",
			yaml: "queue_mapping:\n  0: 0\n  1: 4\n  2: 2\n  3: 6\n",
			want: "ist kleiner als der Eintrag davor",
		},
		{
			name: "festzeitgruen ueber hoechstgruen",
			yaml: "fixed:\n  green_ms: 45000\n",
			want: "ueberschreitet timing.max_green_ms",
		},
		{
			name: "glaettungsfaktor ausserhalb des bereichs",
			yaml: "adaptive:\n  demand_alpha: 1.4\n",
			want: "adaptive.demand_alpha muss zwischen null",
		},
		{
			name: "puffer null",
			yaml: "logging:\n  buffer: 0\n",
			want: "logging.buffer muss groesser als null sein",
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

// TestValidateCollectsAllErrors belegt, dass nicht nur der erste Verstoss gemeldet wird.
func TestValidateCollectsAllErrors(t *testing.T) {
	cfg := Default()
	cfg.Hardware.Chip = ""
	cfg.Logging.Dir = ""
	err := cfg.Validate()
	if err == nil {
		t.Fatal("zwei Verstoesse wurden nicht gemeldet")
	}
	for _, want := range []string{"hardware.chip", "logging.dir"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Fehler %q enthaelt nicht %q", err, want)
		}
	}
}

func bitOrderYAML(north ...string) string {
	bits := append([]string{}, north...)
	bits = append(bits,
		"E_red", "E_yellow", "E_green",
		"S_red", "S_yellow", "S_green",
		"W_red", "W_yellow", "W_green",
		"free", "free", "free", "free",
	)
	return "hardware:\n  shift_register:\n    bit_order: [" + strings.Join(bits, ", ") + "]\n"
}
