import sys

from . import display, driver
from .signal import DIRECTIONS


def open_strip(settings):
    lamps = settings["lamps"]
    return driver.Strip(
        driver.Spi(lamps["spi"], lamps["speed_hz"]),
        len(DIRECTIONS), lamps["brightness"], lamps["pixels"],
    )


def open_screen(settings, lines):
    shown = settings["display"]
    if not shown["enabled"]:
        return None, None
    rotation = driver.Tft.LANDSCAPE
    if shown["rotation"] == "hoch":
        rotation = driver.Tft.PORTRAIT
    bus = None
    dc = None
    try:
        bus = driver.Spi(shown["spi"], shown["speed_hz"])
        dc = lines.output(shown["dc"])
        panel = driver.Tft(bus, dc, rotation)
    except Exception as error:
        if bus:
            bus.close()
        if dc:
            lines.release(dc)
        print(f"Anzeige nicht verfuegbar, Steuerung laeuft ohne: {error}", file=sys.stderr)
        return None, None
    return display.Screen(panel), panel
