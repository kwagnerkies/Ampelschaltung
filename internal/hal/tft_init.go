package hal

import (
	"fmt"
	"time"
)

// Die Startfolge stammt aus dem Datenblatt des ILI9341. Die Register ohne sprechenden Namen
// stellen Spannungen und Zeitverhalten des Panels ein; ihre Werte sind die des Herstellers
// und werden nicht veraendert.
var tftInit = []struct {
	command byte
	data    []byte
	pause   time.Duration
}{
	{command: tftSoftReset, pause: 150 * time.Millisecond},
	{command: 0xCF, data: []byte{0x00, 0xC1, 0x30}},
	{command: 0xED, data: []byte{0x64, 0x03, 0x12, 0x81}},
	{command: 0xE8, data: []byte{0x85, 0x00, 0x78}},
	{command: 0xCB, data: []byte{0x39, 0x2C, 0x00, 0x34, 0x02}},
	{command: 0xF7, data: []byte{0x20}},
	{command: 0xEA, data: []byte{0x00, 0x00}},
	{command: 0xC0, data: []byte{0x23}},
	{command: 0xC1, data: []byte{0x10}},
	{command: 0xC5, data: []byte{0x3E, 0x28}},
	{command: 0xC7, data: []byte{0x86}},
	{command: tftPixelFormat, data: []byte{0x55}},
	{command: 0xB1, data: []byte{0x00, 0x18}},
	{command: 0xB6, data: []byte{0x08, 0x82, 0x27}},
	{command: 0xF2, data: []byte{0x00}},
	{command: 0x26, data: []byte{0x01}},
	{command: 0xE0, data: []byte{0x0F, 0x31, 0x2B, 0x0C, 0x0E, 0x08, 0x4E, 0xF1,
		0x37, 0x07, 0x10, 0x03, 0x0E, 0x09, 0x00}},
	{command: 0xE1, data: []byte{0x00, 0x0E, 0x14, 0x03, 0x11, 0x07, 0x31, 0xC1,
		0x48, 0x08, 0x0F, 0x0C, 0x31, 0x36, 0x0F}},
	{command: tftSleepOut, pause: 120 * time.Millisecond},
	{command: tftDisplayOn, pause: 20 * time.Millisecond},
}

// init bringt den Controller in einen bekannten Zustand. Ohne Reset-Leitung genuegt der
// Software-Reset, die meisten Module ziehen RESET ueber einen Widerstand hoch.
func (t *TFT) init(rotation int) error {
	if t.reset != nil {
		for _, step := range []struct {
			high  bool
			pause time.Duration
		}{{true, 5 * time.Millisecond}, {false, 20 * time.Millisecond}, {true, 150 * time.Millisecond}} {
			if err := t.reset.Set(step.high); err != nil {
				return fmt.Errorf("anzeige: reset-leitung: %w", err)
			}
			time.Sleep(step.pause)
		}
	}
	for _, step := range tftInit {
		if err := t.command(step.command, step.data...); err != nil {
			return err
		}
		if step.pause > 0 {
			time.Sleep(step.pause)
		}
	}
	return t.command(tftMemoryAcces, byte(rotation))
}
