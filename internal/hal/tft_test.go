package hal

import (
	"testing"
)

type tape struct {
	commands []byte
	data     []byte
	writes   int
	dcHigh   bool
}

func (t *tape) Write(data []byte) error {
	t.writes++
	if t.dcHigh {
		t.data = append(t.data, data...)
		return nil
	}
	t.commands = append(t.commands, data...)
	return nil
}

func (t *tape) Close() error { return nil }

type line struct {
	tape *tape
	dc   bool
}

func (l line) Set(high bool) error {
	if l.dc {
		l.tape.dcHigh = high
	}
	return nil
}

func (l line) Close() error { return nil }

func newTFT(t *testing.T, rotation int) (*TFT, *tape) {
	t.Helper()
	tp := &tape{}
	tft, err := NewTFT(tp, line{tape: tp, dc: true}, nil, rotation)
	if err != nil {
		t.Fatalf("NewTFT: %v", err)
	}
	return tft, tp
}

func TestInitWakesAndSwitchesOn(t *testing.T) {
	_, tp := newTFT(t, RotationLandscape)
	for _, want := range []byte{tftSoftReset, tftSleepOut, tftDisplayOn, tftPixelFormat, tftMemoryAcces} {
		if !contains(tp.commands, want) {
			t.Errorf("Befehl %#x fehlt in der Startfolge", want)
		}
	}
	if !contains(tp.data, 0x55) {
		t.Error("das Pixelformat wurde nicht auf 16 Bit gesetzt")
	}
}

func TestRotationDecidesSize(t *testing.T) {
	quer, _ := newTFT(t, RotationLandscape)
	if w, h := quer.Size(); w != 320 || h != 240 {
		t.Errorf("quer ergibt %dx%d, erwartet 320x240", w, h)
	}
	hoch, _ := newTFT(t, RotationPortrait)
	if w, h := hoch.Size(); w != 240 || h != 320 {
		t.Errorf("hochkant ergibt %dx%d, erwartet 240x320", w, h)
	}
}

func TestFillSetsWindowAndWritesEveryPixel(t *testing.T) {
	tft, tp := newTFT(t, RotationLandscape)
	tp.commands, tp.data = nil, nil

	if err := tft.Fill(10, 20, 30, 40, 0xF800); err != nil {
		t.Fatalf("Fill: %v", err)
	}
	if got, want := string(tp.commands), string([]byte{tftColumnAddr, tftPageAddr, tftMemoryWrite}); got != want {
		t.Errorf("Befehlsfolge %#x, erwartet Spalte, Seite, Schreiben", tp.commands)
	}
	pixels := tp.data[8:]
	if got, want := len(pixels), 2*30*40; got != want {
		t.Fatalf("%d Bytes Bilddaten, erwartet %d", got, want)
	}
	for i := 0; i < len(pixels); i += 2 {
		if pixels[i] != 0xF8 || pixels[i+1] != 0x00 {
			t.Fatalf("Bildpunkt %d ist %#x%#x, erwartet F800", i/2, pixels[i], pixels[i+1])
		}
	}
	window := tp.data[:8]
	if window[1] != 10 || window[3] != 39 || window[5] != 20 || window[7] != 59 {
		t.Errorf("Bereich %v, erwartet Spalten 10 bis 39 und Seiten 20 bis 59", window)
	}
}

func TestFillRejectsRectangleOutsideScreen(t *testing.T) {
	tft, _ := newTFT(t, RotationLandscape)
	for _, box := range [][4]int{{300, 0, 30, 10}, {0, 230, 10, 30}, {-1, 0, 10, 10}} {
		if err := tft.Fill(box[0], box[1], box[2], box[3], 0); err == nil {
			t.Errorf("Rechteck %v wurde angenommen", box)
		}
	}
}

func TestEmptyRectangleDrawsNothing(t *testing.T) {
	tft, tp := newTFT(t, RotationLandscape)
	tp.writes = 0
	if err := tft.Fill(0, 0, 0, 10, 0); err != nil {
		t.Fatalf("Fill: %v", err)
	}
	if tp.writes != 0 {
		t.Errorf("%d Schreibzugriffe fuer ein leeres Rechteck", tp.writes)
	}
}

func contains(haystack []byte, needle byte) bool {
	for _, b := range haystack {
		if b == needle {
			return true
		}
	}
	return false
}
