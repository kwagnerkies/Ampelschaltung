// Paket logging schreibt Messdaten als CSV mit Semikolon als Trennzeichen.
package logging

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"sync/atomic"
)

// Writer nimmt Zeilen ueber einen gepufferten Kanal auf und schreibt sie in einer eigenen
// Goroutine. Damit blockiert kein Dateizugriff den Regelkreis. Ist der Puffer voll, wird die
// Zeile verworfen und gezaehlt.
type Writer struct {
	rows    chan []string
	done    chan struct{}
	dropped atomic.Uint64
	closed  atomic.Bool
	err     atomic.Pointer[error]
}

func NewWriter(path string, header []string, buffer int) (*Writer, error) {
	if buffer <= 0 {
		buffer = 1
	}
	file, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("logdatei %s anlegen: %w", path, err)
	}
	return newWriter(file, header, buffer), nil
}

func newWriter(target io.WriteCloser, header []string, buffer int) *Writer {
	if buffer <= 0 {
		buffer = 1
	}
	w := &Writer{
		rows: make(chan []string, buffer),
		done: make(chan struct{}),
	}
	go w.loop(target, header)
	return w
}

// Write reiht eine Zeile ein. Der Aufruf blockiert nie.
func (w *Writer) Write(row []string) {
	if w.closed.Load() {
		return
	}
	select {
	case w.rows <- row:
	default:
		w.dropped.Add(1)
	}
}

// Dropped ist die Anzahl verworfener Zeilen.
func (w *Writer) Dropped() uint64 { return w.dropped.Load() }

// Close leert den Puffer und schliesst die Datei.
func (w *Writer) Close() error {
	if w.closed.Swap(true) {
		return nil
	}
	close(w.rows)
	<-w.done
	if err := w.err.Load(); err != nil {
		return *err
	}
	return nil
}

func (w *Writer) loop(target io.WriteCloser, header []string) {
	defer close(w.done)
	out := csv.NewWriter(target)
	out.Comma = ';'
	if err := out.Write(header); err != nil {
		w.fail(fmt.Errorf("kopfzeile schreiben: %w", err))
	}
	for row := range w.rows {
		if err := out.Write(row); err != nil {
			w.fail(fmt.Errorf("zeile schreiben: %w", err))
			break
		}
	}
	out.Flush()
	if err := out.Error(); err != nil {
		w.fail(fmt.Errorf("puffer leeren: %w", err))
	}
	if err := target.Close(); err != nil {
		w.fail(fmt.Errorf("logdatei schliessen: %w", err))
	}
}

func (w *Writer) fail(err error) {
	if w.err.Load() == nil {
		w.err.Store(&err)
	}
}
