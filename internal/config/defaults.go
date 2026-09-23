package config

import "time"

// Default liefert die Startwerte aus dem Plan. Sie gelten, wenn keine Konfigurationsdatei
// vorhanden ist, und fuellen einzelne fehlende Felder einer vorhandenen Datei.
func Default() Config {
	return Config{
		Hardware: Hardware{
			Chip: "gpiochip0",
			Lamps: Lamps{
				North: [3]int{17, 27, 22},
				East:  [3]int{5, 6, 13},
				South: [3]int{19, 26, 12},
				West:  [3]int{16, 20, 21},
			},
			Sensors: Sensors{
				North: 23,
				East:  24,
				South: 25,
				West:  8,
			},
			PowerSwitch: 4,
			FaultSwitch: 18,
			Debounce:    millis(15),
		},
		Timing: Timing{
			Yellow:    millis(3000),
			RedYellow: millis(1000),
			AllRed:    millis(2000),
			BaseGreen: millis(5000),
			MaxGreen:  millis(20000),
			Follow:    millis(2000),
			Extension: millis(3000),
		},
		Display: Display{
			Enabled:  true,
			Device:   "/dev/spidev0.0",
			SpeedHz:  24000000,
			DC:       7,
			Reset:    2,
			Rotation: "quer",
		},
	}
}

func millis(ms int) Millis {
	return Millis(time.Duration(ms) * time.Millisecond)
}
