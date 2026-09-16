// Programm ampel steuert die Modellkreuzung.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"syscall"

	"ampel/internal/config"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Fehler:", err)
		os.Exit(1)
	}
}

func run() error {
	path := flag.String("config", "configs/config.yaml", "Pfad zur Konfigurationsdatei")
	validate := flag.Bool("validate", false, "Konfiguration pruefen und beenden")
	selftest := flag.Bool("selftest", false, "Lampen und Sensoren pruefen, Abbruch mit Strg-C")
	logDir := flag.String("logdir", "", "Logverzeichnis, ueberschreibt die Konfiguration")
	mode := flag.String("modus", "festzeit", "Betriebsart festzeit oder adaptiv, gilt nur bis der Kippschalter gelesen ist")
	flag.Parse()

	cfg, source, err := loadConfig(*path)
	if err != nil {
		return err
	}
	if *validate {
		printSummary(os.Stdout, cfg, source)
		return nil
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if *selftest {
		return runSelftest(ctx, cfg, os.Stdout)
	}
	return runControl(ctx, cfg, *logDir, *mode, os.Stdout)
}

// loadConfig liefert zusaetzlich die Herkunft der Werte, damit ein fehlender Pfad in der
// Ausgabe sichtbar wird und nicht als geprueft durchgeht.
func loadConfig(path string) (*config.Config, string, error) {
	cfg, err := config.Load(path)
	switch {
	case err == nil:
		return cfg, path, nil
	case errors.Is(err, fs.ErrNotExist):
		fallback := config.Default()
		if err := fallback.Validate(); err != nil {
			return nil, "", fmt.Errorf("eingebaute Defaults sind ungueltig: %w", err)
		}
		return &fallback, "eingebaute Defaults, " + path + " fehlt", nil
	default:
		return nil, "", err
	}
}
