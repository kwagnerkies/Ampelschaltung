package hal

import (
	"errors"
	"reflect"
	"testing"
)

// harness protokolliert, welcher Datenpegel bei welcher Schiebeflanke anlag.
type harness struct {
	data, clock, latch *fakeLine
	level              bool
	captured           []bool
	latchedAfter       []int
}

type fakeLine struct {
	h    *harness
	role string
	high bool
	err  error
	shut bool
}

func newHarness() *harness {
	h := &harness{}
	h.data = &fakeLine{h: h, role: "data"}
	h.clock = &fakeLine{h: h, role: "clock"}
	h.latch = &fakeLine{h: h, role: "latch"}
	return h
}

func (h *harness) driver(bits int) *ShiftRegister {
	return NewShiftRegister(h.data, h.clock, h.latch, bits)
}

func (l *fakeLine) Set(high bool) error {
	if l.err != nil {
		return l.err
	}
	rising := high && !l.high
	l.high = high
	switch {
	case l.role == "data":
		l.h.level = high
	case l.role == "clock" && rising:
		l.h.captured = append(l.h.captured, l.h.level)
	case l.role == "latch" && rising:
		l.h.latchedAfter = append(l.h.latchedAfter, len(l.h.captured))
	}
	return nil
}

func (l *fakeLine) Close() error {
	l.shut = true
	return nil
}

// Das erste Bit des Musters muss zuerst ausgeschoben werden, sonst landet die
// Bitreihenfolge aus der Konfiguration spiegelbildlich auf der Kette.
func TestWriteShiftsFirstBitFirst(t *testing.T) {
	h := newHarness()
	pattern := []bool{true, false, false, true, true, false, false, false}

	if err := h.driver(len(pattern)).Write(pattern); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if !reflect.DeepEqual(h.captured, pattern) {
		t.Errorf("ausgeschoben %v, erwartet %v", h.captured, pattern)
	}
}

// Uebernommen wird erst, wenn alle Bits in der Kette stehen. Ein zu frueher Latch macht
// Zwischenzustaende sichtbar.
func TestWriteLatchesAfterLastBit(t *testing.T) {
	h := newHarness()
	pattern := make([]bool, 16)

	if err := h.driver(len(pattern)).Write(pattern); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if want := []int{16}; !reflect.DeepEqual(h.latchedAfter, want) {
		t.Errorf("Uebernahme nach %v geschobenen Bits, erwartet %v", h.latchedAfter, want)
	}
}

func TestWriteRejectsWrongLength(t *testing.T) {
	h := newHarness()
	err := h.driver(16).Write(make([]bool, 12))
	if err == nil {
		t.Fatal("kurzes Muster wurde angenommen")
	}
	if len(h.captured) != 0 {
		t.Errorf("es wurden %d Bits geschoben, erwartet keines", len(h.captured))
	}
}

func TestWriteReportsLineError(t *testing.T) {
	h := newHarness()
	broken := errors.New("leitung defekt")
	h.clock.err = broken

	err := h.driver(4).Write([]bool{true, false, true, false})
	if !errors.Is(err, broken) {
		t.Fatalf("Fehler %v, erwartet %v", err, broken)
	}
}

// Close muss die Ausgaenge abschalten, bevor es die Leitungen freigibt.
func TestCloseClearsAndReleases(t *testing.T) {
	h := newHarness()
	driver := h.driver(4)

	if err := driver.Write([]bool{true, true, true, true}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := driver.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if got, want := h.captured, []bool{true, true, true, true, false, false, false, false}; !reflect.DeepEqual(got, want) {
		t.Errorf("ausgeschoben %v, erwartet %v", got, want)
	}
	for _, line := range []*fakeLine{h.data, h.clock, h.latch} {
		if !line.shut {
			t.Errorf("leitung %s wurde nicht freigegeben", line.role)
		}
	}
}

func TestClearWritesZeros(t *testing.T) {
	h := newHarness()
	if err := h.driver(3).Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if got, want := h.captured, []bool{false, false, false}; !reflect.DeepEqual(got, want) {
		t.Errorf("ausgeschoben %v, erwartet %v", got, want)
	}
}
