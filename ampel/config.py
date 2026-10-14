import copy
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
    config = copy.deepcopy(DEFAULTS)
    if path is not None:
        try:
            with open(path, "rb") as handle:
                loaded = tomllib.load(handle)
        except FileNotFoundError:
            raise ConfigError(f"{path} nicht gefunden")
        except tomllib.TOMLDecodeError as error:
            raise ConfigError(f"{path}: {error}")
        for section in loaded:
            if section not in config:
                raise ConfigError(f"unbekannter Abschnitt {section}")
            for key in loaded[section]:
                if key not in config[section]:
                    raise ConfigError(f"unbekannter Schluessel {section}.{key}")
                config[section][key] = loaded[section][key]
    validate(config)
    return config


def is_integer(value):
    return type(value) is int


def is_number(value):
    return type(value) is int or type(value) is float


def claim(used, pin, name):
    if not is_integer(pin):
        raise ConfigError(f"{name}: {pin!r} ist keine Pinnummer")
    if pin < 0 or pin > 27:
        raise ConfigError(f"{name}: BCM {pin} liegt ausserhalb von 0 bis 27")
    if pin in used:
        raise ConfigError(f"BCM {pin} ist doppelt belegt: {used[pin]} und {name}")
    used[pin] = name


def validate(config):
    used = {}
    for name in config["sensors"]:
        claim(used, config["sensors"][name], f"sensors.{name}")
    for name in config["switches"]:
        claim(used, config["switches"][name], f"switches.{name}")
    for pin in SPI_PINS:
        claim(used, pin, SPI_PINS[pin])
    if config["display"]["enabled"]:
        claim(used, config["display"]["dc"], "display.dc")

    pixels = config["lamps"]["pixels"]
    if len(pixels) != 3:
        raise ConfigError(f"lamps.pixels {pixels}: drei verschiedene Werte von 0 bis 7")
    for pixel in pixels:
        if not is_integer(pixel) or pixel < 0 or pixel > 7 or pixels.count(pixel) > 1:
            raise ConfigError(f"lamps.pixels {pixels}: drei verschiedene Werte von 0 bis 7")

    brightness = config["lamps"]["brightness"]
    if not is_integer(brightness) or brightness < 0 or brightness > 255:
        raise ConfigError(f"lamps.brightness {brightness!r}: ganze Zahl von 0 bis 255")

    for section in ["lamps", "display"]:
        speed = config[section]["speed_hz"]
        if not is_integer(speed) or speed <= 0:
            raise ConfigError(f"{section}.speed_hz {speed!r}: ganze Zahl groesser 0")

    for section in ["display", "api"]:
        if type(config[section]["enabled"]) is not bool:
            raise ConfigError(f"{section}.enabled: true oder false")

    rotation = config["display"]["rotation"]
    if rotation not in ("quer", "hoch"):
        raise ConfigError(f"display.rotation {rotation!r}: quer oder hoch")

    timing = config["timing"]
    for name in timing:
        if not is_number(timing[name]) or timing[name] <= 0:
            raise ConfigError(f"timing.{name} {timing[name]!r}: Zahl groesser 0")
    if timing["max_green"] < timing["base_green"]:
        raise ConfigError("timing.max_green ist kleiner als timing.base_green")
