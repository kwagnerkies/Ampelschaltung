package config

import (
	"ampel/internal/controller"
	"ampel/internal/hal"
	"ampel/internal/strategy"
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Hardware Hardware `yaml:"hardware"`
	Timing   Timing   `yaml:"timing"`
	Display  Display  `yaml:"display"`
	API      API      `yaml:"api"`
}

type Hardware struct {
	Chip        string  `yaml:"chip"`
	Lamps       Lamps   `yaml:"lamps"`
	Sensors     Sensors `yaml:"sensors"`
	PowerSwitch int     `yaml:"power_switch"`
	FaultSwitch int     `yaml:"fault_switch"`
	Debounce    Millis  `yaml:"debounce_ms"`
}

type Lamps struct {
	North [3]int `yaml:"north"`
	East  [3]int `yaml:"east"`
	South [3]int `yaml:"south"`
	West  [3]int `yaml:"west"`
}

func (l Lamps) Heads() [4][3]int {
	return [4][3]int{l.North, l.East, l.South, l.West}
}

type Sensors struct {
	North int `yaml:"north"`
	East  int `yaml:"east"`
	South int `yaml:"south"`
	West  int `yaml:"west"`
}

type Timing struct {
	Yellow    Millis `yaml:"yellow_ms"`
	RedYellow Millis `yaml:"red_yellow_ms"`
	AllRed    Millis `yaml:"all_red_ms"`
	BaseGreen Millis `yaml:"base_green_ms"`
	MaxGreen  Millis `yaml:"max_green_ms"`
	Follow    Millis `yaml:"follow_ms"`
	Extension Millis `yaml:"extension_ms"`
}

type API struct {
	Enabled bool   `yaml:"enabled"`
	Address string `yaml:"address"`
}

type Display struct {
	Enabled  bool   `yaml:"enabled"`
	Device   string `yaml:"spi"`
	SpeedHz  int    `yaml:"speed_hz"`
	DC       int    `yaml:"dc"`
	Reset    int    `yaml:"reset"`
	Rotation string `yaml:"rotation"`
}

func (s Sensors) Approaches() [4]int {
	return [4]int{s.North, s.East, s.South, s.West}
}

func (t Timing) Intergreen() time.Duration {
	return t.Yellow.Duration() + t.AllRed.Duration() + t.RedYellow.Duration()
}

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
				West:  3,
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
		API: API{Enabled: true, Address: ":8080"},
		Display: Display{
			Enabled:  true,
			Device:   "/dev/spidev0.0",
			SpeedHz:  24000000,
			DC:       2,
			Reset:    -1,
			Rotation: "quer",
		},
	}
}

func millis(ms int) Millis {
	return Millis(time.Duration(ms) * time.Millisecond)
}

type Millis time.Duration

func (m *Millis) UnmarshalYAML(node *yaml.Node) error {
	var ms int64
	if err := node.Decode(&ms); err != nil {
		return fmt.Errorf("ganzzahlige millisekunden erwartet: %w", err)
	}
	*m = Millis(time.Duration(ms) * time.Millisecond)
	return nil
}

func (m Millis) MarshalYAML() (any, error) {
	return time.Duration(m).Milliseconds(), nil
}

func (m Millis) Duration() time.Duration { return time.Duration(m) }

func (m Millis) String() string { return time.Duration(m).String() }

func (c *Config) ControllerTiming() controller.Timing {
	return controller.Timing{
		Yellow:    c.Timing.Yellow.Duration(),
		AllRed:    c.Timing.AllRed.Duration(),
		RedYellow: c.Timing.RedYellow.Duration(),
	}
}

func (c *Config) Following() (*strategy.Following, error) {
	return strategy.NewFollowing(
		c.Timing.BaseGreen.Duration(),
		c.Timing.Extension.Duration(),
		c.Timing.MaxGreen.Duration(),
	)
}

func (d Display) TFTRotation() int {
	if d.Rotation == "hoch" {
		return hal.RotationPortrait
	}
	return hal.RotationLandscape
}

func (c *Config) Setup() (controller.Setup, error) {
	return controller.Setup{
		Sensors: c.Hardware.Sensors.Approaches(),
		Timing:  c.ControllerTiming(),
		Follow:  c.Timing.Follow.Duration(),
		Tick:    controller.DefaultTick,
		Sample:  time.Second,
	}, nil
}
