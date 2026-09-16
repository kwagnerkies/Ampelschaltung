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
	fmt.Fprintf(w, "  Bedienelemente     Kippschalter %d, Reset-Taster %d, Entprellung %s\n",
		cfg.Hardware.ModeSwitch, cfg.Hardware.ResetButton, cfg.Hardware.Debounce)
	fmt.Fprintf(w, "  Zwischenzeiten     Gelb %s, Allrot %s, RotGelb %s, Summe %s\n",
		cfg.Timing.Yellow, cfg.Timing.AllRed, cfg.Timing.RedYellow, cfg.Timing.Intergreen())
	fmt.Fprintf(w, "  Gruenzeiten        min %s, max %s, Umlauf %s, davon verteilbar %s\n",
		cfg.Timing.MinGreen, cfg.Timing.MaxGreen, cfg.Timing.Cycle, cfg.Timing.CycleEffective())
	fmt.Fprintf(w, "  Anforderung        Luecke %s, Verlaengerung %s, Hoechstwartezeit %s\n",
		cfg.Timing.Gap, cfg.Timing.Extension, cfg.Timing.MaxWait)
	fmt.Fprintf(w, "  Festzeitbetrieb    Gruen %s\n", cfg.Fixed.Green)
	fmt.Fprintf(w, "  Adaptiv            demand_alpha %v, learn_alpha %v, blend_k %v\n",
		cfg.Adaptive.DemandAlpha, cfg.Adaptive.LearnAlpha, cfg.Adaptive.BlendK)
	fmt.Fprintf(w, "  Rueckstautabelle   %s\n", queueMappingText(cfg.QueueMapping, cfg.Hardware.Sensors.SensorCount()))
	fmt.Fprintf(w, "  Logging            %s, Abtastung %s, Puffer %d\n",
		cfg.Logging.Dir, cfg.Logging.StateInterval, cfg.Logging.Buffer)
	fmt.Fprintf(w, "  Lernzustand        %s, Sicherung alle %s\n", cfg.Learning.Path, cfg.Learning.SaveInterval)
}

func queueMappingText(mapping map[int]int, sensorCount int) string {
	text := ""
	for occupied := 0; occupied <= sensorCount; occupied++ {
		if occupied > 0 {
			text += ", "
		}
		text += fmt.Sprintf("%d belegt = %d Fahrzeuge", occupied, mapping[occupied])
	}
	return text
}
