package ws2812

import (
	"testing"
)

type tape struct {
	frames [][]byte
}

func (t *tape) Write(data []byte) error {
	frame := make([]byte, len(data))
	copy(frame, data)
	t.frames = append(t.frames, frame)
	return nil
}

func (t *tape) Close() error { return nil }

func newStrip(t *testing.T, brightness byte) (*Strip, *tape) {
	t.Helper()
	tp := &tape{}
	strip, err := New(tp, 4, brightness)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return strip, tp
}

func bits(frame []byte, pixel int) []byte {
	start := pixel * bitsPerPixel * spiPerBit / 8
	return frame[start : start+bitsPerPixel*spiPerBit/8]
}

func colorOf(frame []byte, pixel int) (green, red, blue byte) {
	raw := bits(frame, pixel)
	var value [3]byte
	index := 0
	for byteIndex := 0; byteIndex < 3; byteIndex++ {
		for mask := byte(0x80); mask > 0; mask >>= 1 {
			pattern := 0
			for offset := 0; offset < spiPerBit; offset++ {
				if raw[index/8]&(0x80>>(index%8)) != 0 {
					pattern |= 1 << (spiPerBit - 1 - offset)
				}
				index++
			}
			if pattern == patternOne {
				value[byteIndex] |= mask
			}
		}
	}
	return value[0], value[1], value[2]
}

func TestEachHeadUsesItsOwnPixels(t *testing.T) {
	strip, tp := newStrip(t, 255)
	lamps := make([]bool, 12)
	lamps[0] = true
	lamps[4] = true
	lamps[8] = true
	if err := strip.Write(lamps); err != nil {
		t.Fatalf("Write: %v", err)
	}

	frame := tp.frames[0]
	cases := map[int][3]byte{
		0*PixelsPerHead + Pixels[0]: {0, 255, 0},
		1*PixelsPerHead + Pixels[1]: {150, 255, 0},
		2*PixelsPerHead + Pixels[2]: {255, 0, 0},
	}
	for pixel, want := range cases {
		green, red, blue := colorOf(frame, pixel)
		if [3]byte{green, red, blue} != want {
			t.Errorf("Pixel %d ist %v, erwartet %v", pixel, [3]byte{green, red, blue}, want)
		}
	}
}

func TestDarkLampsStayBlack(t *testing.T) {
	strip, tp := newStrip(t, 255)
	if err := strip.Write(make([]bool, 12)); err != nil {
		t.Fatalf("Write: %v", err)
	}
	for pixel := 0; pixel < 4*PixelsPerHead; pixel++ {
		if green, red, blue := colorOf(tp.frames[0], pixel); green|red|blue != 0 {
			t.Fatalf("Pixel %d leuchtet mit %d %d %d", pixel, green, red, blue)
		}
	}
}

func TestBrightnessScalesTheColour(t *testing.T) {
	strip, tp := newStrip(t, 51)
	lamps := make([]bool, 12)
	lamps[0] = true
	if err := strip.Write(lamps); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if _, red, _ := colorOf(tp.frames[0], Pixels[0]); red != 51 {
		t.Errorf("Rot mit %d Helligkeit ergibt %d, erwartet 51", 51, red)
	}
}

func TestFrameEndsWithAReset(t *testing.T) {
	strip, tp := newStrip(t, 255)
	lamps := make([]bool, 12)
	lamps[0] = true
	if err := strip.Write(lamps); err != nil {
		t.Fatalf("Write: %v", err)
	}
	frame := tp.frames[0]
	for _, b := range frame[len(frame)-resetBytes:] {
		if b != 0 {
			t.Fatal("die Ruhephase am Ende ist nicht null")
		}
	}
}

func TestWrongNumberOfLampsIsRejected(t *testing.T) {
	strip, _ := newStrip(t, 255)
	if err := strip.Write(make([]bool, 11)); err == nil {
		t.Error("elf Lampenzustaende wurden angenommen")
	}
}
