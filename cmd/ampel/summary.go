package main

import (
	"fmt"
	"io"

	"ampel/internal/config"
)

func printSummary(w io.Writer, cfg *config.Config, source string) {
	fmt.Fprintf(w, "Konfiguration in Ordnung (%s)\n", source)
	fmt.Fprintf(w, "  GPIO-Chip          %s\n", cfg.Hardware.Chip)
	fmt.Fprintf(w, "  Schieberegister    Data %d, Clock %d, Latch %d\n",
		cfg.Hardware.ShiftRegister.Data, cfg.Hardware.ShiftRegister.Clock, cfg.Hardware.ShiftRegister.Latch)
	names := [4]string{"Nord", "Ost", "Sued", "West"}
	for i, pins := range cfg.Hardware.Sensors.Approaches() {
		fmt.Fprintf(w, "  Sensoren %-5s     %v\n", names[i], pins)
	}
	fmt.Fprintf(w, "  Hauptschalter      BCM %d, Entprellung %s\n",
		cfg.Hardware.PowerSwitch, cfg.Hardware.Debounce)
	fmt.Fprintf(w, "  Zwischenzeiten     Gelb %s, Allrot %s, RotGelb %s, Summe %s\n",
		cfg.Timing.Yellow, cfg.Timing.AllRed, cfg.Timing.RedYellow, cfg.Timing.Intergreen())
	fmt.Fprintf(w, "  Gruenzeiten        Grundzeit %s, hoechstens %s\n",
		cfg.Timing.BaseGreen, cfg.Timing.MaxGreen)
	fmt.Fprintf(w, "  Verlaengerung      %s je Fahrzeug, das binnen %s folgt\n",
		cfg.Timing.Extension, cfg.Timing.Follow)
	fmt.Fprintf(w, "  Festzeitbetrieb    Gruen %s\n", cfg.Fixed.Green)
	if cfg.Display.Enabled {
		fmt.Fprintf(w, "  Anzeige            %s, %d Hz, DC %d, Reset %d, %s\n",
			cfg.Display.Device, cfg.Display.SpeedHz, cfg.Display.DC, cfg.Display.Reset, cfg.Display.Rotation)
	} else {
		fmt.Fprintln(w, "  Anzeige            abgeschaltet")
	}
	fmt.Fprintf(w, "  Logging            %s, Abtastung %s, Puffer %d\n",
		cfg.Logging.Dir, cfg.Logging.StateInterval, cfg.Logging.Buffer)
}
