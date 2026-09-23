package main

import (
	"fmt"
	"io"

	"ampel/internal/config"
)

func printSummary(w io.Writer, cfg *config.Config, source string) {
	fmt.Fprintf(w, "Konfiguration in Ordnung (%s)\n", source)
	fmt.Fprintf(w, "  GPIO-Chip          %s\n", cfg.Hardware.Chip)
	for i, head := range cfg.Hardware.Lamps.Heads() {
		fmt.Fprintf(w, "  Lampen %-5s       Rot %d, Gelb %d, Gruen %d\n",
			approachNames[i], head[0], head[1], head[2])
	}
	for i, pin := range cfg.Hardware.Sensors.Approaches() {
		fmt.Fprintf(w, "  Haltelinie %-5s   BCM %d\n", approachNames[i], pin)
	}
	fmt.Fprintf(w, "  Schalter           Hauptschalter BCM %d, Notschalter BCM %d, Entprellung %s\n",
		cfg.Hardware.PowerSwitch, cfg.Hardware.FaultSwitch, cfg.Hardware.Debounce)
	fmt.Fprintf(w, "  Zwischenzeiten     Gelb %s, Allrot %s, RotGelb %s, Summe %s\n",
		cfg.Timing.Yellow, cfg.Timing.AllRed, cfg.Timing.RedYellow, cfg.Timing.Intergreen())
	fmt.Fprintf(w, "  Gruenzeiten        Grundzeit %s, hoechstens %s\n",
		cfg.Timing.BaseGreen, cfg.Timing.MaxGreen)
	fmt.Fprintf(w, "  Verlaengerung      %s je Fahrzeug, das binnen %s folgt\n",
		cfg.Timing.Extension, cfg.Timing.Follow)
	if cfg.Display.Enabled {
		fmt.Fprintf(w, "  Anzeige            %s, %d Hz, DC %d, Reset %d, %s\n",
			cfg.Display.Device, cfg.Display.SpeedHz, cfg.Display.DC, cfg.Display.Reset, cfg.Display.Rotation)
	} else {
		fmt.Fprintln(w, "  Anzeige            abgeschaltet")
	}
}
