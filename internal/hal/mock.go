package hal

import (
	"fmt"
	"slices"
	"sync"
	"time"
)

// Mock ersetzt Lampentreiber und Eingaenge in Tests und im Simulator.
type Mock struct {
	mu      sync.Mutex
	pattern []bool
	history [][]bool
	closed  bool
	levels  map[int]bool
	events  chan InputEvent
}

var (
	_ LampDriver  = (*Mock)(nil)
	_ InputSource = (*Mock)(nil)
)

func NewMock(bits, buffer int) *Mock {
	return &Mock{
		pattern: make([]bool, bits),
		levels:  make(map[int]bool),
		events:  make(chan InputEvent, buffer),
	}
}

func (m *Mock) Write(pattern []bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(pattern) != len(m.pattern) {
		return fmt.Errorf("muster hat %d bits, die kette hat %d", len(pattern), len(m.pattern))
	}
	copy(m.pattern, pattern)
	m.history = append(m.history, slices.Clone(pattern))
	return nil
}

func (m *Mock) Clear() error {
	return m.Write(make([]bool, len(m.pattern)))
}

func (m *Mock) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *Mock) Events() <-chan InputEvent { return m.events }

func (m *Mock) Read(pin int) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.levels[pin], nil
}

// Pattern liefert eine Kopie des zuletzt geschriebenen Musters.
func (m *Mock) Pattern() []bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.pattern)
}

// History liefert alle geschriebenen Muster in ihrer Reihenfolge. Damit pruefen Tests
// vollstaendige Signalfolgen und nicht nur den Endzustand.
func (m *Mock) History() [][]bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	copied := make([][]bool, len(m.history))
	for i, pattern := range m.history {
		copied[i] = slices.Clone(pattern)
	}
	return copied
}

func (m *Mock) Writes() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.history)
}

func (m *Mock) Closed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}

// SetLevel legt den Pegel fuer Read fest, ohne eine Flanke zu melden. Fuer den Kippschalter.
func (m *Mock) SetLevel(pin int, active bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.levels[pin] = active
}

// Emit meldet eine Flanke und setzt den Pegel passend nach. Blockiert nie: ist der Kanal
// voll, laeuft der Test in einen Fehler statt in einen Deadlock.
func (m *Mock) Emit(pin int, active bool, at time.Time) error {
	m.SetLevel(pin, active)
	select {
	case m.events <- InputEvent{Pin: pin, Active: active, Time: at}:
		return nil
	default:
		return fmt.Errorf("ereigniskanal voll, flanke an BCM %d verworfen", pin)
	}
}
