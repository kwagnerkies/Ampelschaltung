package spi

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"

	"ampel/src/driver"
)

const (
	spiWriteMode        = 0x40016B01
	spiWriteBitsPerWord = 0x40016B03
	spiWriteMaxSpeedHz  = 0x40046B04
)

type SPI struct {
	file *os.File
}

var _ driver.Transport = (*SPI)(nil)

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
