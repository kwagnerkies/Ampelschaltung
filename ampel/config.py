import tomllib

DEFAULTS = {
    "lamps": {"spi": "/dev/spidev1.0", "speed_hz": 2400000, "brightness": 60, "pixels": [0, 4, 7]},
    "sensors": {"north": 23, "east": 24, "south": 25, "west": 3},
    "switches": {"power": 4, "fault": 27},
    "display": {"enabled": True, "spi": "/dev/spidev0.0", "speed_hz": 24000000, "dc": 2, "rotation": "quer"},
    "api": {"enabled": True, "socket": "/run/ampel/ampel.sock"},
    "timing": {
        "yellow": 3.0, "all_red": 2.0, "red_yellow": 1.0,
        "base_green": 5.0, "max_green": 20.0, "follow": 2.0, "extension": 3.0,
    },
}

SPI_PINS = {7: "spi0 ce1", 8: "spi0 ce0", 9: "spi0 miso", 10: "spi0 mosi", 11: "spi0 sclk",
            18: "spi1 ce0", 19: "spi1 miso", 20: "spi1 mosi", 21: "spi1 sclk"}


class ConfigError(Exception):
    pass


def load(path=None):
    config = {section: dict(values) for section, values in DEFAULTS.items()}
    if path:
        try:
            with open(path, "rb") as handle:
                for section, values in tomllib.load(handle).items():
                    if section not in config:
                        raise ConfigError(f"unbekannter Abschnitt {section}")
                    config[section].update(values)
        except FileNotFoundError:
            pass
    validate(config)
    return config


def validate(config):
    used = {}
    def claim(pin, name):
        if not 0 <= pin <= 27:
            raise ConfigError(f"{name}: BCM {pin} liegt ausserhalb von 0 bis 27")
        if pin in used:
            raise ConfigError(f"BCM {pin} ist doppelt belegt: {used[pin]} und {name}")
        used[pin] = name

    for name, pin in config["sensors"].items():
        claim(pin, f"sensors.{name}")
    for name, pin in config["switches"].items():
        claim(pin, f"switches.{name}")
    for pin, name in SPI_PINS.items():
        claim(pin, name)
    if config["display"]["enabled"]:
        claim(config["display"]["dc"], "display.dc")

    pixels = config["lamps"]["pixels"]
    if len(pixels) != 3 or len(set(pixels)) != 3 or not all(0 <= p <= 7 for p in pixels):
        raise ConfigError(f"lamps.pixels {pixels}: drei verschiedene Werte von 0 bis 7")
