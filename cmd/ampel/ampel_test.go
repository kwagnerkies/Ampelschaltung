package main

import (
	"ampel/internal/config"
	"ampel/internal/controller"
	"ampel/internal/hal"
	"ampel/internal/light"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWalkLampsLightsEachLampAlone(t *testing.T) {
	cfg := config.Default()
	driver := hal.NewMock(controller.LampCount, 1)

	if err := walkLamps(context.Background(), io.Discard, driver, &cfg, 0); err != nil {
		t.Fatalf("walkLamps: %v", err)
	}

	history := driver.History()
	if len(history) != controller.LampCount+1 {
		t.Fatalf("%d Schreibzugriffe, erwartet %d", len(history), controller.LampCount+1)
	}
	for lamp := 0; lamp < controller.LampCount; lamp++ {
		for index, on := range history[lamp] {
			if on != (index == lamp) {
				t.Errorf("bei lampe %d ist %d %v, erwartet %v", lamp, index, on, index == lamp)
			}
		}
	}
	for index, on := range history[controller.LampCount] {
		if on {
			t.Errorf("nach dem Test leuchtet noch Lampe %d", index)
		}
	}
}

func TestInputPinsCoversSensorsAndSwitches(t *testing.T) {
	cfg := config.Default()
	pins, labels := inputPins(&cfg)

	if len(pins) != 6 {
		t.Fatalf("%d Eingaenge, erwartet 6", len(pins))
	}
	for _, pin := range pins {
		if labels[pin] == "" {
			t.Errorf("BCM %d hat keine Bezeichnung", pin)
		}
	}
	if got := labels[cfg.Hardware.Sensors.North]; got != "Nord Haltelinie" {
		t.Errorf("Bezeichnung %q, erwartet \"Nord Haltelinie\"", got)
	}
	if got := labels[cfg.Hardware.FaultSwitch]; got != "Notschalter" {
		t.Errorf("Bezeichnung %q, erwartet \"Notschalter\"", got)
	}
}

func TestSequencePassesGuardedOutput(t *testing.T) {
	cfg := config.Default()
	mock := hal.NewMock(controller.LampCount, 1)
	output := controller.NewOutput(mock)

	greens := map[light.Direction]int{}
	for i, s := range sequenceSteps(cfg.Timing) {
		if err := output.Show(s.aspects); err != nil {
			t.Fatalf("schritt %d (%s): %v", i, describe(s.aspects), err)
		}
		for direction, aspect := range s.aspects {
			if aspect == light.AspectGreen {
				greens[light.Direction(direction)]++
			}
		}
	}

	for _, direction := range light.Directions() {
		if greens[direction] != 1 {
			t.Errorf("zufahrt %s war %d mal gruen, erwartet einmal", direction, greens[direction])
		}
	}
}

func TestSequenceWrapsGreenInTransitions(t *testing.T) {
	cfg := config.Default()
	steps := sequenceSteps(cfg.Timing)

	for _, direction := range light.Directions() {
		var seen []light.Aspect
		for _, s := range steps {
			aspect := s.aspects[direction]
			if len(seen) == 0 || seen[len(seen)-1] != aspect {
				seen = append(seen, aspect)
			}
		}
		green := -1
		for i, aspect := range seen {
			if aspect == light.AspectGreen {
				green = i
			}
		}
		if green <= 0 || green+1 >= len(seen) {
			t.Fatalf("zufahrt %s: gruen liegt nicht innerhalb der folge %v", direction, seen)
		}
		if seen[green-1] != light.AspectRedYellow {
			t.Errorf("zufahrt %s zeigt vor Gruen %s, erwartet RotGelb", direction, seen[green-1])
		}
		if seen[green+1] != light.AspectYellow {
			t.Errorf("zufahrt %s zeigt nach Gruen %s, erwartet Gelb", direction, seen[green+1])
		}
	}
}

func TestSequenceHoldsComeFromConfig(t *testing.T) {
	cfg := config.Default()
	cfg.Timing.BaseGreen = config.Millis(4 * time.Second)
	steps := sequenceSteps(cfg.Timing)

	for _, s := range steps {
		var want time.Duration
		switch {
		case containsAspect(s.aspects, light.AspectGreen):
			want = 4 * time.Second
		case containsAspect(s.aspects, light.AspectYellow):
			want = cfg.Timing.Yellow.Duration()
		case containsAspect(s.aspects, light.AspectRedYellow):
			want = cfg.Timing.RedYellow.Duration()
		default:
			want = cfg.Timing.AllRed.Duration()
		}
		if s.hold != want {
			t.Errorf("%s haelt %s, erwartet %s", describe(s.aspects), s.hold, want)
		}
	}
}

func containsAspect(aspects [light.DirectionCount]light.Aspect, want light.Aspect) bool {
	for _, aspect := range aspects {
		if aspect == want {
			return true
		}
	}
	return false
}

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

func directive(unit, key string) string {
	values := directives(unit, key)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func directives(unit, key string) []string {
	var values []string
	for _, line := range strings.Split(unit, "\n") {
		if name, value, found := strings.Cut(strings.TrimSpace(line), "="); found && name == key {
			values = append(values, strings.TrimSpace(value))
		}
	}
	return values
}

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
