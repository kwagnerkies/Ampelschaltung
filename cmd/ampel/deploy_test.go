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
	values := directives(unit, key)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

// directives liefert alle Werte eines mehrfach erlaubten Schluessels.
func directives(unit, key string) []string {
	var values []string
	for _, line := range strings.Split(unit, "\n") {
		if name, value, found := strings.Cut(strings.TrimSpace(line), "="); found && name == key {
			values = append(values, strings.TrimSpace(value))
		}
	}
	return values
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

// Die Geraetefreigabe der Unit muss zu der Konfiguration passen, sonst startet der Dienst und
// scheitert erst beim ersten Zugriff.
func TestUnitMatchesConfig(t *testing.T) {
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Konfiguration laden: %v", err)
	}
	unit := read(t, unitPath)

	allowed := directives(unit, "DeviceAllow")
	wanted := []string{"/dev/" + cfg.Hardware.Chip + " rw"}
	if cfg.Display.Enabled {
		wanted = append(wanted, cfg.Display.Device+" rw")
	}
	for _, want := range wanted {
		if !contains(allowed, want) {
			t.Errorf("DeviceAllow %v enthaelt %q nicht", allowed, want)
		}
	}
}

// Das Installationsskript muss genau die Pfade anlegen, die Dienst und Konfiguration
// erwarten.
func TestInstallScriptCreatesRequiredPaths(t *testing.T) {
	script := read(t, installPath)

	for _, path := range []string{
		filepath.Dir(installedConf),
		installedBin,
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
