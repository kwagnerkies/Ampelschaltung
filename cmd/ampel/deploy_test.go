package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ampel/internal/config"
)

const (
	unitPath      = "../../deploy/ampel.service"
	installPath   = "../../deploy/install.sh"
	configPath    = "../../configs/config.yaml"
	installedBin  = "/usr/local/bin/ampel"
	installedConf = "/etc/ampel/config.yaml"
)

func read(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%s lesen: %v", path, err)
	}
	return string(content)
}

// directive liefert den Wert einer Zeile der Unit.
func directive(unit, key string) string {
	for _, line := range strings.Split(unit, "\n") {
		if name, value, found := strings.Cut(strings.TrimSpace(line), "="); found && name == key {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// Der Dienst muss das installierte Programm mit der installierten Konfiguration starten und
// nach einem Absturz von selbst wiederkommen.
func TestUnitStartsInstalledProgram(t *testing.T) {
	unit := read(t, unitPath)

	start := directive(unit, "ExecStart")
	if !strings.HasPrefix(start, installedBin+" ") {
		t.Errorf("ExecStart ist %q, erwartet den Start von %s", start, installedBin)
	}
	if !strings.Contains(start, "-config "+installedConf) {
		t.Errorf("ExecStart ist %q, erwartet die Konfiguration %s", start, installedConf)
	}
	want := map[string]string{
		"Restart":      "always",
		"RestartSec":   "2",
		"User":         "ampel",
		"Group":        "gpio",
		"KillSignal":   "SIGTERM",
		"WantedBy":     "multi-user.target",
		"DevicePolicy": "closed",
	}
	for key, value := range want {
		if got := directive(unit, key); got != value {
			t.Errorf("%s ist %q, erwartet %q", key, got, value)
		}
	}
}

// Schreibrechte, Geraetefreigabe und Verzeichnisse muessen zu der Konfiguration passen, die
// installiert wird. Sonst startet der Dienst und scheitert erst beim ersten Schreibversuch.
func TestUnitMatchesConfig(t *testing.T) {
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Konfiguration laden: %v", err)
	}
	unit := read(t, unitPath)

	writable := strings.Fields(directive(unit, "ReadWritePaths"))
	for _, path := range []string{cfg.Logging.Dir, filepath.Dir(cfg.Learning.Path)} {
		if !contains(writable, path) {
			t.Errorf("ReadWritePaths %v enthaelt %s nicht", writable, path)
		}
	}
	if got, want := directive(unit, "DeviceAllow"), "/dev/"+cfg.Hardware.Chip+" rw"; got != want {
		t.Errorf("DeviceAllow ist %q, erwartet %q", got, want)
	}
	if got := directive(unit, "ProtectSystem"); got == "strict" && len(writable) == 0 {
		t.Error("ProtectSystem=strict ohne ReadWritePaths macht das Logverzeichnis unbeschreibbar")
	}
}

// Das Installationsskript muss genau die Pfade anlegen, die Dienst und Konfiguration
// erwarten.
func TestInstallScriptCreatesRequiredPaths(t *testing.T) {
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Konfiguration laden: %v", err)
	}
	script := read(t, installPath)

	for _, path := range []string{
		cfg.Logging.Dir,
		filepath.Dir(cfg.Learning.Path),
		filepath.Dir(installedConf),
		installedBin,
		// Die Auswertung muss mitkommen, die Vorfuehrung zeigt die Kennzahlen auf dem Pi.
		"/usr/local/bin/ampeleval",
	} {
		if !strings.Contains(script, path) {
			t.Errorf("install.sh legt %s nicht an", path)
		}
	}
	for _, step := range []string{"useradd", "systemctl daemon-reload", "systemctl enable --now", "-validate"} {
		if !strings.Contains(script, step) {
			t.Errorf("install.sh fuehrt %q nicht aus", step)
		}
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
