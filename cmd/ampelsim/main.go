// Programm ampelsim fuehrt den Regelkreis ohne Hardware mit erzeugtem Verkehr aus.
package main

import (
	"errors"
	"flag"
	"fmt"
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
	duration := flag.Duration("dauer", 10*time.Minute, "simulierte Dauer")
	seed := flag.Int64("seed", 1, "Startwert des Zufallsgenerators")
	// Die Grundlast liegt bewusst unter der Saettigung. Darueber steht mehr in der Zufahrt,
	// als die Sensoren sehen.
	rates := flag.String("raten", "0.08,0.03,0.08,0.03", "Ankuenfte pro Sekunde fuer Nord,Ost,Sued,West")
	startClock := flag.String("start", "07:00", "Startzeit der Simulation")
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

	s, err := newSimulation(simOptions{config: cfg, seed: *seed, rates: parsedRates, start: start})
	if err != nil {
		return err
	}
	var show func(*simulation)
	if *display {
		show = func(s *simulation) { render(os.Stdout, s) }
	}
	s.walk(*duration, show)
	if err := s.close(); err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "\nSimulierte Dauer %s\n%s\n", *duration, s.result())
	return nil
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

// parseRates liest vier Ankunftsraten in der Reihenfolge Nord, Ost, Sued, West.
func parseRates(text string) ([light.DirectionCount]float64, error) {
	var rates [light.DirectionCount]float64
	parts := strings.Split(text, ",")
	if len(parts) != light.DirectionCount {
		return rates, fmt.Errorf("vier Raten erwartet, %d angegeben", len(parts))
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

// parseStart liest die Startzeit als Stunde und Minute.
func parseStart(text string) (time.Time, error) {
	parsed, err := time.Parse("15:04", text)
	if err != nil {
		return time.Time{}, fmt.Errorf("startzeit %q: %w", text, err)
	}
	return time.Date(2026, 1, 5, parsed.Hour(), parsed.Minute(), 0, 0, time.UTC), nil
}
