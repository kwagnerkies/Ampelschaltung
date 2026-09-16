package config

import "time"

// Default liefert die Startwerte aus dem Plan. Sie gelten, wenn keine Konfigurationsdatei
// vorhanden ist, und fuellen einzelne fehlende Felder einer vorhandenen Datei.
func Default() Config {
	return Config{
		Hardware: Hardware{
			Chip: "gpiochip0",
			ShiftRegister: ShiftRegister{
				Data:  17,
				Clock: 27,
				Latch: 22,
				BitOrder: []string{
					"N_red", "N_yellow", "N_green",
					"E_red", "E_yellow", "E_green",
					"S_red", "S_yellow", "S_green",
					"W_red", "W_yellow", "W_green",
					"free", "free", "free", "free",
				},
			},
			Sensors: Sensors{
				North: []int{5, 6, 13},
				East:  []int{19, 26, 12},
				South: []int{16, 20, 21},
				West:  []int{23, 24, 25},
			},
			ModeSwitch:  4,
			ResetButton: 18,
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
		Fixed:        Fixed{Green: millis(15000)},
		QueueMapping: map[int]int{0: 0, 1: 1, 2: 3, 3: 6},
		Logging: Logging{
			Dir:           "/var/log/ampel",
			StateInterval: millis(1000),
			Buffer:        4096,
		},
	}
}

func millis(ms int) Millis {
	return Millis(time.Duration(ms) * time.Millisecond)
}
