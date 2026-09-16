// Programm ampelsim fuehrt den Regelkreis ohne Hardware mit erzeugtem Verkehr aus.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strconv"
	"strings"
	"time"

	"ampel/internal/config"
	"ampel/internal/light"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Fehler:", err)
		os.Exit(1)
	}
}

func run() error {
	configPath := flag.String("config", "configs/config.yaml", "Pfad zur Konfigurationsdatei")
	mode := flag.String("modus", "vergleich", "festzeit, adaptiv oder vergleich")
	duration := flag.Duration("dauer", 30*time.Minute, "simulierte Dauer")
	seed := flag.Int64("seed", 1, "Startwert des Zufallsgenerators")
	// Die Grundlast liegt bewusst unter der Saettigung. Darueber steht mehr im Rueckstau als
	// die drei Sensoren je Zufahrt sehen, und die gemessene Wartezeit wird unbrauchbar.
	rates := flag.String("raten", "0.08,0.03,0.08,0.03", "Ankuenfte pro Sekunde fuer Nord,Ost,Sued,West")
	amplitude := flag.Float64("tagesgang", 0, "Staerke des Tagesgangs zwischen 0 und 1")
	startClock := flag.String("start", "07:00", "Startzeit der Simulation")
	logDir := flag.String("logdir", "", "Logverzeichnis fuer die CSV-Dateien")
	display := flag.Bool("anzeige", false, "Kreuzung im Terminal anzeigen")
	flag.Parse()

	cfg, err := loadConfig(*configPath)
	if err != nil {
		return err
	}
	parsedRates, err := parseRates(*rates)
	if err != nil {
		return err
	}
	start, err := parseStart(*startClock)
	if err != nil {
		return err
	}

	options := simOptions{
		config:    cfg,
		seed:      *seed,
		rates:     parsedRates,
		amplitude: *amplitude,
		start:     start,
		logDir:    *logDir,
	}

	modes := []string{*mode}
	if *mode == "vergleich" {
		modes = []string{"festzeit", "adaptiv"}
	}

	var results []result
	for _, name := range modes {
		options.mode = name
		r, err := simulate(options, *duration, *display)
		if err != nil {
			return err
		}
		results = append(results, r)
	}
	printResults(os.Stdout, results, *duration)
	return nil
}

func simulate(options simOptions, duration time.Duration, display bool) (result, error) {
	s, err := newSimulation(options)
	if err != nil {
		return result{}, err
	}
	var show func(*simulation)
	if display {
		show = func(s *simulation) { render(os.Stdout, s) }
	}
	s.walk(duration, show)
	if err := s.close(); err != nil {
		return result{}, err
	}
	return s.result(), nil
}

func printResults(out io.Writer, results []result, duration time.Duration) {
	fmt.Fprintf(out, "\nSimulierte Dauer %s\n", duration)
	for _, r := range results {
		fmt.Fprintln(out, r)
	}
	if len(results) != 2 {
		return
	}
	first, second := results[0], results[1]
	if first.truth == 0 || second.truth == 0 {
		return
	}
	better, worse := first, second
	if second.truth < first.truth {
		better, worse = second, first
	}
	share := 100 * (1 - float64(better.truth)/float64(worse.truth))
	fmt.Fprintf(out, "\n%s liegt %.1f Prozent unter %s (%s gegen %s).\n",
		better.mode, share, worse.mode, round(better.truth), round(worse.truth))
}

func loadConfig(path string) (*config.Config, error) {
	cfg, err := config.Load(path)
	if err == nil {
		return cfg, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	fallback := config.Default()
	if err := fallback.Validate(); err != nil {
		return nil, err
	}
	return &fallback, nil
}

func parseRates(text string) ([light.DirectionCount]float64, error) {
	var rates [light.DirectionCount]float64
	parts := strings.Split(text, ",")
	if len(parts) != light.DirectionCount {
		return rates, fmt.Errorf("raten brauchen vier Werte fuer Nord,Ost,Sued,West, nicht %d", len(parts))
	}
	for i, part := range parts {
		value, err := strconv.ParseFloat(strings.TrimSpace(part), 64)
		if err != nil {
			return rates, fmt.Errorf("rate %q: %w", part, err)
		}
		if value < 0 {
			return rates, fmt.Errorf("rate %v ist negativ", value)
		}
		rates[i] = value
	}
	return rates, nil
}

// parseStart legt den Startzeitpunkt der simulierten Zeit fest. Das Datum ist beliebig, nur
// die Tageszeit zaehlt fuer den Tagesgang.
func parseStart(text string) (time.Time, error) {
	parsed, err := time.Parse("15:04", text)
	if err != nil {
		return time.Time{}, fmt.Errorf("startzeit %q: %w", text, err)
	}
	return time.Date(2026, 1, 5, parsed.Hour(), parsed.Minute(), 0, 0, time.UTC), nil
}
