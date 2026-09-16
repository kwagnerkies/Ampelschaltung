package main

import (
	"encoding/csv"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Row ist eine Zeile aus vehicles.csv.
type Row struct {
	Mode      string
	Direction string
	Wait      time.Duration
	Queue     int
	Phase     string
	Settling  bool
}

// readPaths liest Dateien und Verzeichnisse ein. In einem Verzeichnis werden alle Dateien
// beruecksichtigt, deren Name mit vehicles beginnt.
func readPaths(paths []string) ([]Row, error) {
	var rows []Row
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		if !info.IsDir() {
			found, err := readFile(path)
			if err != nil {
				return nil, err
			}
			rows = append(rows, found...)
			continue
		}
		entries, err := os.ReadDir(path)
		if err != nil {
			return nil, fmt.Errorf("%s lesen: %w", path, err)
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasPrefix(entry.Name(), "vehicles") {
				continue
			}
			found, err := readFile(filepath.Join(path, entry.Name()))
			if err != nil {
				return nil, err
			}
			rows = append(rows, found...)
		}
	}
	if len(rows) == 0 {
		return nil, fs.ErrNotExist
	}
	return rows, nil
}

func readFile(path string) ([]Row, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("%s oeffnen: %w", path, err)
	}
	defer func() { _ = file.Close() }()

	reader := csv.NewReader(file)
	reader.Comma = ';'
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("%s lesen: %w", path, err)
	}
	if len(records) == 0 {
		return nil, nil
	}

	index := columnIndex(records[0])
	var rows []Row
	for number, record := range records[1:] {
		row, err := parseRow(record, index)
		if err != nil {
			return nil, fmt.Errorf("%s zeile %d: %w", path, number+2, err)
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func columnIndex(header []string) map[string]int {
	index := make(map[string]int, len(header))
	for position, name := range header {
		index[strings.TrimSpace(name)] = position
	}
	return index
}

func parseRow(record []string, index map[string]int) (Row, error) {
	field := func(name string) string {
		position, ok := index[name]
		if !ok || position >= len(record) {
			return ""
		}
		return record[position]
	}
	wait, err := strconv.ParseInt(field("wartezeit_ms"), 10, 64)
	if err != nil {
		return Row{}, fmt.Errorf("wartezeit_ms: %w", err)
	}
	queue, err := strconv.Atoi(field("rueckstau_bei_ankunft"))
	if err != nil {
		queue = 0
	}
	return Row{
		Mode:      field("modus"),
		Direction: field("zufahrt"),
		Wait:      time.Duration(wait) * time.Millisecond,
		Queue:     queue,
		Phase:     field("phase_bei_ankunft"),
		Settling:  field("einschwingen") == "1",
	}, nil
}
