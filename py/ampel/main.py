import argparse
import signal as unix_signal
import sys
import time

from . import api, config, display, driver
from .control import Controller, Timing
from .rule import Following
from .signal import DIRECTIONS, DIRECTION_NAMES

TICK = 0.05


def main(argv=None):
    parser = argparse.ArgumentParser(prog="ampel")
    parser.add_argument("-config", default="/etc/ampel/config.toml")
    parser.add_argument("-validate", action="store_true")
    parser.add_argument("-selftest", action="store_true")
    args = parser.parse_args(argv)

    settings = config.load(args.config)
    if args.validate:
        summary(settings)
        return 0
    if args.selftest:
        return selftest(settings)
    return run(settings)


def summary(settings):
    print("Konfiguration in Ordnung")
    lamps = settings["lamps"]
    print(f"  Lampen        {lamps['spi']}, Helligkeit {lamps['brightness']}, Pixel {lamps['pixels']}")
    for name, pin in settings["sensors"].items():
        print(f"  Haltelinie    {name:<6} BCM {pin}")
    print(f"  Schalter      Haupt BCM {settings['switches']['power']}, "
          f"Not BCM {settings['switches']['fault']}")
    timing = settings["timing"]
    print(f"  Zwischenzeit  Gelb {timing['yellow']}s, Allrot {timing['all_red']}s, "
          f"RotGelb {timing['red_yellow']}s")
    print(f"  Gruenzeit     Grundzeit {timing['base_green']}s, hoechstens {timing['max_green']}s")
    print(f"  Verlaengerung {timing['extension']}s je Fahrzeug binnen {timing['follow']}s")


def build(settings, lines):
    lamps = settings["lamps"]
    strip = driver.Strip(
        driver.Spi(lamps["spi"], lamps["speed_hz"]),
        len(DIRECTIONS), lamps["brightness"], lamps["pixels"],
    )
    timing = settings["timing"]
    rule = Following(timing["base_green"], timing["extension"], timing["max_green"])
    controller = Controller(strip, rule,
                            Timing(timing["yellow"], timing["all_red"], timing["red_yellow"]),
                            timing["follow"], time.monotonic())
    return controller, strip


def open_screen(settings, lines):
    shown = settings["display"]
    if not shown["enabled"]:
        return None, None
    panel = driver.Tft(
        driver.Spi(shown["spi"], shown["speed_hz"]),
        lines.output(shown["dc"]),
        driver.Tft.PORTRAIT if shown["rotation"] == "hoch" else driver.Tft.LANDSCAPE,
    )
    return display.Screen(panel), panel


def run(settings):
    from .gpio import Lines
    lines = Lines()
    controller, strip = build(settings, lines)
    screen, panel = open_screen(settings, lines)

    sensors = {pin: direction for direction, pin in
               zip(DIRECTIONS, [settings["sensors"][name.lower()] for name in
                                ("north", "east", "south", "west")])}
    power_pin = settings["switches"]["power"]
    fault_pin = settings["switches"]["fault"]

    def edge(pin, closed):
        now = time.monotonic()
        if pin in sensors:
            if not closed:
                controller.crossing(now, sensors[pin])
        elif pin == power_pin:
            controller.power(now, closed)
        elif pin == fault_pin:
            controller.warn(now, closed)

    lines.watch(list(sensors) + [power_pin, fault_pin], edge)
    controller.power(time.monotonic(), lines.read(power_pin) or True)

    server = None
    if settings["api"]["enabled"]:
        server = api.serve(
            settings["api"]["socket"],
            lambda: api.status_of(controller.snapshot(time.monotonic())),
            lambda switch, on: edge(power_pin if switch == "hauptschalter" else fault_pin, on),
        )

    stopping = []
    unix_signal.signal(unix_signal.SIGTERM, lambda *_: stopping.append(True))
    unix_signal.signal(unix_signal.SIGINT, lambda *_: stopping.append(True))

    print("Betrieb gestartet")
    try:
        while not stopping:
            now = time.monotonic()
            controller.step(now)
            if screen:
                screen.update(display.fields_from(controller.snapshot(now)))
            time.sleep(TICK)
    finally:
        controller.shutdown()
        if server:
            server.shutdown()
        if panel:
            panel.close()
        strip.close()
        lines.close()
    return 0


def selftest(settings):
    from .gpio import Lines
    lines = Lines()
    controller, strip = build(settings, lines)
    screen, panel = open_screen(settings, lines)

    if screen:
        print("Anzeigetest: vier mal 88")
        screen.update({d: (88, display.GREEN) for d in DIRECTIONS})
    print("Lampentest, jede Lampe einzeln:")
    for index in range(len(DIRECTIONS) * 3):
        lamps = [False] * (len(DIRECTIONS) * 3)
        lamps[index] = True
        strip.write(lamps)
        print(f"  Stick {index // 3} Pixel {settings['lamps']['pixels'][index % 3]}  "
              f"{DIRECTION_NAMES[index // 3]} {('Rot', 'Gelb', 'Gruen')[index % 3]}")
        time.sleep(0.4)
    strip.write([False] * 12)

    print("Sensortest, Fahrzeug ueber die Kontakte schieben. Abbruch mit Strg-C.")
    lines.watch(list(settings["sensors"].values()) + list(settings["switches"].values()),
                lambda pin, closed: print(f"  BCM {pin} {'geschlossen' if closed else 'offen'}"))
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


if __name__ == "__main__":
    sys.exit(main())
