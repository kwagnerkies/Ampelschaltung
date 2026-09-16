package logging

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriterWritesHeaderAndRows(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.csv")
	w, err := NewWriter(path, []string{"a", "b"}, 8)
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	w.Write([]string{"1", "2"})
	w.Write([]string{"3", "4"})
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	records := read(t, path)
	want := [][]string{{"a", "b"}, {"1", "2"}, {"3", "4"}}
	if len(records) != len(want) {
		t.Fatalf("%d Zeilen, erwartet %d", len(records), len(want))
	}
	for i, row := range want {
		if strings.Join(records[i], ";") != strings.Join(row, ";") {
			t.Errorf("zeile %d ist %v, erwartet %v", i, records[i], row)
		}
	}
	if w.Dropped() != 0 {
		t.Errorf("%d Zeilen verworfen", w.Dropped())
	}
}

// Ist der Puffer voll, muss die Zeile verworfen und gezaehlt werden. Der Regelkreis darf
// niemals auf die Platte warten.
func TestWriterDropsWhenBufferFull(t *testing.T) {
	blocked := &blockingTarget{release: make(chan struct{})}
	w := newWriter(blocked, []string{"a"}, 1)

	for i := 0; i < 50; i++ {
		w.Write([]string{"x"})
	}
	if w.Dropped() == 0 {
		t.Error("keine Zeile verworfen, obwohl der Puffer voll war")
	}

	close(blocked.release)
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestWriteAfterCloseIsIgnored(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.csv")
	w, err := NewWriter(path, []string{"a"}, 4)
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	w.Write([]string{"x"})
	if err := w.Close(); err != nil {
		t.Errorf("zweites Close: %v", err)
	}
}

// blockingTarget haelt den Schreiber auf, bis release geschlossen wird.
type blockingTarget struct {
	release chan struct{}
}

func (b *blockingTarget) Write(p []byte) (int, error) {
	<-b.release
	return len(p), nil
}

func (b *blockingTarget) Close() error { return nil }

func read(t *testing.T, path string) [][]string {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("oeffnen: %v", err)
	}
	defer func() { _ = file.Close() }()
	reader := csv.NewReader(file)
	reader.Comma = ';'
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("lesen: %v", err)
	}
	return records
}
