package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"time"
)

var directionOrder = []string{"Nord", "Ost", "Sued", "West"}

// report gibt die Kennzahlen als Tabelle aus.
func report(out io.Writer, modes map[string][]time.Duration, perDirection map[string]map[string][]time.Duration, includeSettling bool) {
	names := sortedKeys(modes)

	if includeSettling {
		fmt.Fprintln(out, "Einschwingphasen sind enthalten.")
	} else {
		fmt.Fprintln(out, "Einschwingphasen nach einem Moduswechsel sind ausgeschlossen.")
	}
	fmt.Fprintf(out, "\n%-12s %10s %10s %10s %10s %10s\n", "Modus", "Fahrzeuge", "Mittel", "Median", "P95", "Maximum")
	for _, mode := range names {
		printStats(out, mode, compute(modes[mode]))
	}

	fmt.Fprintln(out, "\nAufschluesselung nach Zufahrt")
	fmt.Fprintf(out, "%-12s %-6s %10s %10s %10s %10s\n", "Modus", "Zufahrt", "Fahrzeuge", "Mittel", "Median", "Maximum")
	for _, mode := range names {
		for _, direction := range directionsIn(perDirection[mode]) {
			stats := compute(perDirection[mode][direction])
			fmt.Fprintf(out, "%-12s %-6s %10d %10s %10s %10s\n", mode, direction,
				stats.Count, seconds(stats.Mean), seconds(stats.Median), seconds(stats.Max))
		}
	}

	if len(names) == 2 {
		first, second := compute(modes[names[0]]), compute(modes[names[1]])
		printComparison(out, names[0], first, names[1], second)
	}
}

func printStats(out io.Writer, mode string, stats Stats) {
	fmt.Fprintf(out, "%-12s %10d %10s %10s %10s %10s\n", mode, stats.Count,
		seconds(stats.Mean), seconds(stats.Median), seconds(stats.P95), seconds(stats.Max))
}

func printComparison(out io.Writer, nameA string, a Stats, nameB string, b Stats) {
	if a.Mean == 0 || b.Mean == 0 {
		return
	}
	better, worse, betterName, worseName := a, b, nameA, nameB
	if b.Mean < a.Mean {
		better, worse, betterName, worseName = b, a, nameB, nameA
	}
	share := 100 * (1 - float64(better.Mean)/float64(worse.Mean))
	fmt.Fprintf(out, "\n%s liegt %.1f Prozent unter %s (%s gegen %s).\n",
		betterName, share, worseName, seconds(better.Mean), seconds(worse.Mean))
}

// writeCSV schreibt dieselben Kennzahlen fuer die Ausarbeitung.
func writeCSV(path string, modes map[string][]time.Duration, perDirection map[string]map[string][]time.Duration) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("%s anlegen: %w", path, err)
	}
	defer func() { _ = file.Close() }()

	out := csv.NewWriter(file)
	out.Comma = ';'
	defer out.Flush()

	if err := out.Write([]string{"modus", "zufahrt", "fahrzeuge", "mittel_ms", "median_ms", "p95_ms", "max_ms"}); err != nil {
		return fmt.Errorf("kopfzeile: %w", err)
	}
	for _, mode := range sortedKeys(modes) {
		if err := out.Write(statsRow(mode, "alle", compute(modes[mode]))); err != nil {
			return fmt.Errorf("zeile schreiben: %w", err)
		}
		for _, direction := range directionsIn(perDirection[mode]) {
			if err := out.Write(statsRow(mode, direction, compute(perDirection[mode][direction]))); err != nil {
				return fmt.Errorf("zeile schreiben: %w", err)
			}
		}
	}
	return out.Error()
}

func statsRow(mode, direction string, stats Stats) []string {
	return []string{
		mode, direction, strconv.Itoa(stats.Count),
		strconv.FormatInt(stats.Mean.Milliseconds(), 10),
		strconv.FormatInt(stats.Median.Milliseconds(), 10),
		strconv.FormatInt(stats.P95.Milliseconds(), 10),
		strconv.FormatInt(stats.Max.Milliseconds(), 10),
	}
}

func seconds(d time.Duration) string {
	return strconv.FormatFloat(d.Seconds(), 'f', 2, 64) + " s"
}

func sortedKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

// directionsIn haelt die Reihenfolge Nord, Ost, Sued, West ein und haengt Unbekanntes an.
func directionsIn(m map[string][]time.Duration) []string {
	var ordered []string
	for _, direction := range directionOrder {
		if _, ok := m[direction]; ok {
			ordered = append(ordered, direction)
		}
	}
	for _, direction := range sortedKeys(m) {
		if !slices.Contains(directionOrder, direction) {
			ordered = append(ordered, direction)
		}
	}
	return ordered
}
