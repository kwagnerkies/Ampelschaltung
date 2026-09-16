package hal

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

// ioctl-Nummern des spidev-Treibers. Der Kernel bildet sie aus Richtung, Datengroesse,
// Magic-Byte 'k' und laufender Nummer; x/sys/unix definiert sie nicht.
const (
	spiWriteMode        = 0x40016B01
	spiWriteBitsPerWord = 0x40016B03
	spiWriteMaxSpeedHz  = 0x40046B04
)

// Transport ist ein schreibender Kanal zur Anzeige. Das Interface steht hier, damit der
// Displaytreiber ohne Hardware getestet werden kann.
type Transport interface {
	Write(data []byte) error
	Close() error
}

// SPI ist das Character-Device /dev/spidevX.Y. Geschrieben wird halbduplex, mehr braucht ein
// Display nicht: die Antwort des Controllers interessiert nicht.
type SPI struct {
	file *os.File
}

var _ Transport = (*SPI)(nil)

// OpenSPI stellt Modus 0, acht Bit je Wort und die Taktrate ein.
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

// Write schreibt einen Block. Der Kernel zerlegt zu grosse Bloecke selbst.
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
