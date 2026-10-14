import time

from .. import display
from ..hardware import open_screen, open_strip
from ..signal import DIRECTIONS, DIRECTION_NAMES, EAST, NORTH, SOUTH, WEST

LAMP_NAMES = ["Rot", "Gelb", "Gruen"]


def selftest(settings):
    from .gpio import Lines
    lines = Lines()
    strip = open_strip(settings)
    screen, panel = open_screen(settings, lines)
    lamp_count = len(DIRECTIONS) * 3

    if screen:
        print("Anzeigetest: vier mal 88 in Gruen, Rot, Gelb und Weiss")
        screen.update({NORTH: (88, display.GREEN), EAST: (88, display.RED),
                       SOUTH: (88, display.YELLOW), WEST: (88, display.WHITE)})
    print("Lampentest, jede Lampe einzeln:")
    for direction in DIRECTIONS:
        for lamp in range(3):
            lamps = [False] * lamp_count
            lamps[direction * 3 + lamp] = True
            strip.write(lamps)
            pixel = settings["lamps"]["pixels"][lamp]
            print(f"  Stick {direction} Pixel {pixel}  "
                  f"{DIRECTION_NAMES[direction]} {LAMP_NAMES[lamp]}")
            time.sleep(0.4)
    strip.write([False] * lamp_count)

    def show_edge(pin, closed):
        if closed:
            print(f"  BCM {pin} geschlossen")
        else:
            print(f"  BCM {pin} offen")

    print("Sensortest, Fahrzeug ueber die Kontakte schieben. Abbruch mit Strg-C.")
    pins = list(settings["sensors"].values()) + list(settings["switches"].values())
    lines.watch(pins, show_edge)
    try:
        while True:
            time.sleep(0.2)
    except KeyboardInterrupt:
        pass
    finally:
        if panel:
            panel.close()
        strip.close()
        lines.close()
    return 0
