package tft

import (
	"fmt"
	"os"
	"time"

	"golang.org/x/sys/unix"

	"ampel/src/treiber"
)

const (
	spiWriteMode        = 0x40016B01
	spiWriteBitsPerWord = 0x40016B03
	spiWriteMaxSpeedHz  = 0x40046B04
)

type Transport interface {
	Write(data []byte) error
	Close() error
}

type SPI struct {
	file *os.File
}

var _ Transport = (*SPI)(nil)

func OpenSPI(device string, speedHz int) (*SPI, error) {
	file, err := os.OpenFile(device, os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("spi %s oeffnen: %w", device, err)
	}
	settings := []struct {
		request uint
		value   int
		name    string
	}{
		{spiWriteMode, 0, "modus"},
		{spiWriteBitsPerWord, 8, "wortbreite"},
		{spiWriteMaxSpeedHz, speedHz, "taktrate"},
	}
	for _, s := range settings {
		if err := unix.IoctlSetPointerInt(int(file.Fd()), s.request, s.value); err != nil {
			_ = file.Close()
			return nil, fmt.Errorf("spi %s setzen: %w", s.name, err)
		}
	}
	return &SPI{file: file}, nil
}

func (s *SPI) Write(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	if _, err := s.file.Write(data); err != nil {
		return fmt.Errorf("spi schreiben: %w", err)
	}
	return nil
}

func (s *SPI) Close() error {
	if err := s.file.Close(); err != nil {
		return fmt.Errorf("spi schliessen: %w", err)
	}
	return nil
}

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

const (
	RotationPortrait  = 0x48
	RotationLandscape = 0x28
)

const chunk = 4096

type TFT struct {
	bus           Transport
	dc, reset     treiber.OutputLine
	width, height int
}

func NewTFT(bus Transport, dc, reset treiber.OutputLine, rotation int) (*TFT, error) {
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
