// Programm ampeleval wertet die Fahrzeugdaten eines Laufs aus.
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Fehler:", err)
		os.Exit(1)
	}
}

func run() error {
	includeSettling := flag.Bool("einschwingen", false, "Einschwingphasen nach einem Moduswechsel mitzaehlen")
	csvPath := flag.String("csv", "", "Kennzahlen zusaetzlich als CSV ablegen")
	flag.Parse()

	paths := flag.Args()
	if len(paths) == 0 {
		paths = []string{"/var/log/ampel"}
	}

	rows, err := readPaths(paths)
	if err != nil {
		return err
	}
	modes, perDirection := group(rows, *includeSettling)
	if len(modes) == 0 {
		return fmt.Errorf("keine Fahrzeuge ausserhalb der Einschwingphasen gefunden")
	}
	report(os.Stdout, modes, perDirection, *includeSettling)

	if *csvPath != "" {
		if err := writeCSV(*csvPath, modes, perDirection); err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "\nKennzahlen geschrieben nach %s\n", *csvPath)
	}
	return nil
}
