package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func seconds10(values ...int) []time.Duration {
	waits := make([]time.Duration, 0, len(values))
	for _, value := range values {
		waits = append(waits, time.Duration(value)*time.Second)
	}
	return waits
}

func TestComputeStats(t *testing.T) {
	stats := compute(seconds10(10, 2, 4, 8, 6))

	if stats.Count != 5 {
		t.Errorf("Anzahl %d, erwartet 5", stats.Count)
	}
	if stats.Mean != 6*time.Second {
		t.Errorf("Mittel %s, erwartet 6s", stats.Mean)
	}
	if stats.Median != 6*time.Second {
		t.Errorf("Median %s, erwartet 6s", stats.Median)
	}
	if stats.P95 != 10*time.Second {
		t.Errorf("P95 %s, erwartet 10s", stats.P95)
	}
	if stats.Max != 10*time.Second {
		t.Errorf("Maximum %s, erwartet 10s", stats.Max)
	}
}

func TestComputeEmpty(t *testing.T) {
	if got := compute(nil); got.Count != 0 || got.Mean != 0 {
		t.Errorf("leere Menge ergibt %+v", got)
	}
}

func TestPercentileRanks(t *testing.T) {
	sorted := seconds10(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
	if got := percentile(sorted, 0.5); got != 5*time.Second {
		t.Errorf("Median %s, erwartet 5s", got)
	}
	if got := percentile(sorted, 0.95); got != 10*time.Second {
		t.Errorf("P95 %s, erwartet 10s", got)
	}
}

// Die Einschwingphase wird standardmaessig ausgeschlossen, laesst sich aber zuschalten.
func TestGroupExcludesSettlingByDefault(t *testing.T) {
	rows := []Row{
		{Mode: "festzeit", Direction: "Nord", Wait: 10 * time.Second, Settling: true},
		{Mode: "festzeit", Direction: "Nord", Wait: 4 * time.Second},
		{Mode: "adaptiv", Direction: "Ost", Wait: 2 * time.Second},
	}

	modes, perDirection := group(rows, false)
	if len(modes["festzeit"]) != 1 {
		t.Errorf("%d Fahrzeuge im Festzeitbetrieb, erwartet eines", len(modes["festzeit"]))
	}
	if len(perDirection["adaptiv"]["Ost"]) != 1 {
		t.Errorf("Aufschluesselung fehlt: %v", perDirection)
	}

	modes, _ = group(rows, true)
	if len(modes["festzeit"]) != 2 {
		t.Errorf("%d Fahrzeuge mit Einschwingphase, erwartet zwei", len(modes["festzeit"]))
	}
}

func TestReadPathsFromDirectory(t *testing.T) {
	dir := t.TempDir()
	content := strings.Join([]string{
		"run_id;zeit_iso;t_ms;modus;zufahrt;wartezeit_ms;belegt_bei_ankunft;phase_bei_ankunft;einschwingen",
		"r1;2026-05-02T10:00:00.000Z;0;festzeit;Nord;12000;2;NS;0",
		"r1;2026-05-02T10:00:10.000Z;10000;festzeit;Ost;8000;1;NS;1",
	}, "\n") + "\n"
	if err := os.WriteFile(filepath.Join(dir, "vehicles-r1.csv"), []byte(content), 0o644); err != nil {
		t.Fatalf("schreiben: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "state-r1.csv"), []byte("egal\n"), 0o644); err != nil {
		t.Fatalf("schreiben: %v", err)
	}

	rows, err := readPaths([]string{dir})
	if err != nil {
		t.Fatalf("readPaths: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("%d Zeilen, erwartet zwei", len(rows))
	}
	if rows[0].Wait != 12*time.Second || rows[0].Direction != "Nord" || rows[0].Reach != 2 {
		t.Errorf("erste Zeile %+v", rows[0])
	}
	if !rows[1].Settling {
		t.Error("die zweite Zeile ist nicht als Einschwingphase erkannt")
	}
}

func TestReportComparesTwoModes(t *testing.T) {
	rows := []Row{
		{Mode: "festzeit", Direction: "Nord", Wait: 20 * time.Second},
		{Mode: "festzeit", Direction: "Ost", Wait: 20 * time.Second},
		{Mode: "adaptiv", Direction: "Nord", Wait: 10 * time.Second},
		{Mode: "adaptiv", Direction: "Ost", Wait: 10 * time.Second},
	}
	modes, perDirection := group(rows, false)

	var out bytes.Buffer
	report(&out, modes, perDirection, false)
	text := out.String()

	for _, want := range []string{"festzeit", "adaptiv", "50.0 Prozent", "Nord", "Ost"} {
		if !strings.Contains(text, want) {
			t.Errorf("die Ausgabe enthaelt %q nicht:\n%s", want, text)
		}
	}
}

func TestWriteCSVProducesRows(t *testing.T) {
	rows := []Row{
		{Mode: "festzeit", Direction: "Nord", Wait: 20 * time.Second},
		{Mode: "festzeit", Direction: "Ost", Wait: 10 * time.Second},
	}
	modes, perDirection := group(rows, false)
	path := filepath.Join(t.TempDir(), "kennzahlen.csv")

	if err := writeCSV(path, modes, perDirection); err != nil {
		t.Fatalf("writeCSV: %v", err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("lesen: %v", err)
	}
	text := string(content)
	if !strings.Contains(text, "modus;zufahrt;fahrzeuge") {
		t.Errorf("Kopfzeile fehlt:\n%s", text)
	}
	if !strings.Contains(text, "festzeit;alle;2;15000;10000;20000;20000") {
		t.Errorf("Gesamtzeile fehlt:\n%s", text)
	}
}
