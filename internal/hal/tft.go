package hal

import "fmt"

// Befehle des ILI9341, wie sie im Datenblatt heissen.
const (
	tftSoftReset   = 0x01
	tftSleepOut    = 0x11
	tftDisplayOn   = 0x29
	tftColumnAddr  = 0x2A
	tftPageAddr    = 0x2B
	tftMemoryWrite = 0x2C
	tftMemoryAcces = 0x36
	tftPixelFormat = 0x3A
)

// Drehung der Anzeige. Der Wert landet im Register Memory Access Control und bestimmt
// zugleich, ob 240x320 oder 320x240 gilt.
const (
	RotationPortrait  = 0x48
	RotationLandscape = 0x28
)

// chunk ist der Puffer, mit dem Flaechen gefuellt werden. Groesser bringt nichts, der
// Kernel zerlegt ohnehin.
const chunk = 4096

// TFT ist ein Farbdisplay mit ILI9341 an SPI. Daten und Befehle unterscheidet die
// DC-Leitung, ein Reset-Impuls bringt den Controller in einen bekannten Zustand.
type TFT struct {
	bus           Transport
	dc, reset     OutputLine
	width, height int
}

func NewTFT(bus Transport, dc, reset OutputLine, rotation int) (*TFT, error) {
	if bus == nil || dc == nil {
		return nil, fmt.Errorf("anzeige: bus und dc-leitung sind pflicht")
	}
	t := &TFT{bus: bus, dc: dc, reset: reset, width: 240, height: 320}
	if rotation == RotationLandscape {
		t.width, t.height = 320, 240
	}
	if err := t.init(rotation); err != nil {
		return nil, err
	}
	return t, nil
}

func (t *TFT) Size() (int, int) { return t.width, t.height }

// Fill faerbt ein Rechteck. Die Farbe ist RGB565, also zwei Bytes je Punkt.
func (t *TFT) Fill(x, y, width, height int, color uint16) error {
	if width <= 0 || height <= 0 {
		return nil
	}
	if x < 0 || y < 0 || x+width > t.width || y+height > t.height {
		return fmt.Errorf("anzeige: rechteck %d,%d %dx%d liegt ausserhalb von %dx%d",
			x, y, width, height, t.width, t.height)
	}
	if err := t.window(x, y, width, height); err != nil {
		return err
	}
	high, low := byte(color>>8), byte(color)
	buffer := make([]byte, 0, chunk)
	for len(buffer) < chunk-1 {
		buffer = append(buffer, high, low)
	}
	if err := t.dc.Set(true); err != nil {
		return fmt.Errorf("anzeige: dc auf daten: %w", err)
	}
	remaining := 2 * width * height
	for remaining > 0 {
		block := buffer
		if remaining < len(block) {
			block = block[:remaining]
		}
		if err := t.bus.Write(block); err != nil {
			return err
		}
		remaining -= len(block)
	}
	return nil
}

func (t *TFT) Close() error { return t.bus.Close() }

// window setzt den Schreibbereich und leitet das Schreiben in den Bildspeicher ein.
func (t *TFT) window(x, y, width, height int) error {
	x1, y1 := x+width-1, y+height-1
	if err := t.command(tftColumnAddr, byte(x>>8), byte(x), byte(x1>>8), byte(x1)); err != nil {
		return err
	}
	if err := t.command(tftPageAddr, byte(y>>8), byte(y), byte(y1>>8), byte(y1)); err != nil {
		return err
	}
	return t.command(tftMemoryWrite)
}

func (t *TFT) command(code byte, data ...byte) error {
	if err := t.dc.Set(false); err != nil {
		return fmt.Errorf("anzeige: dc auf befehl: %w", err)
	}
	if err := t.bus.Write([]byte{code}); err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}
	if err := t.dc.Set(true); err != nil {
		return fmt.Errorf("anzeige: dc auf daten: %w", err)
	}
	return t.bus.Write(data)
}
