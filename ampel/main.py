import argparse
import signal as unix_signal
import sys
import time

from . import api, config, display
from .control import Controller, Timing
from .driver.selftest import selftest
from .hardware import open_screen, open_strip
from .rule import Following
from .signal import EAST, NORTH, SOUTH, WEST

TICK = 0.05


def main(argv=None):
    parser = argparse.ArgumentParser(prog="ampel")
    parser.add_argument("-config")
    parser.add_argument("-validate", action="store_true")
    parser.add_argument("-selftest", action="store_true")
    args = parser.parse_args(argv)

    try:
        settings = config.load(args.config)
    except config.ConfigError as error:
        print(f"Konfigurationsfehler: {error}", file=sys.stderr)
        return 1
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
    for name in settings["sensors"]:
        print(f"  Haltelinie    {name:<6} BCM {settings['sensors'][name]}")
    print(f"  Schalter      Haupt BCM {settings['switches']['power']}, "
          f"Not BCM {settings['switches']['fault']}")
    timing = settings["timing"]
    print(f"  Zwischenzeit  Gelb {timing['yellow']}s, Allrot {timing['all_red']}s, "
          f"RotGelb {timing['red_yellow']}s")
    print(f"  Gruenzeit     Grundzeit {timing['base_green']}s, hoechstens {timing['max_green']}s")
    print(f"  Verlaengerung {timing['extension']}s je Fahrzeug binnen {timing['follow']}s")


def build(settings, strip):
    timing = settings["timing"]
    rule = Following(timing["base_green"], timing["extension"], timing["max_green"])
    return Controller(strip, rule,
                      Timing(timing["yellow"], timing["all_red"], timing["red_yellow"]),
                      timing["follow"], time.monotonic())


def draw(screen, controller, now):
    try:
        screen.update(display.fields_from(controller.snapshot(now)))
    except Exception as error:
        print(f"Anzeige ausgefallen, Steuerung laeuft ohne: {error}", file=sys.stderr)
        return None
    return screen


def run(settings):
    from .driver.gpio import Lines
    stopping = []

    def stop(signal_number, frame):
        stopping.append(True)

    unix_signal.signal(unix_signal.SIGTERM, stop)
    unix_signal.signal(unix_signal.SIGINT, stop)

    lines = Lines()
    strip = open_strip(settings)
    controller = build(settings, strip)
    screen = None
    panel = None
    server = None

    pins = settings["sensors"]
    sensors = {pins["north"]: NORTH, pins["east"]: EAST, pins["south"]: SOUTH, pins["west"]: WEST}
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

    def status():
        return api.status_of(controller.snapshot(time.monotonic()))

    def command(switch, on):
        if switch == "hauptschalter":
            edge(power_pin, on)
        else:
            edge(fault_pin, on)

    try:
        screen, panel = open_screen(settings, lines)
        lines.watch(list(sensors) + [power_pin, fault_pin], edge)
        controller.power(time.monotonic(), lines.read(power_pin))
        controller.warn(time.monotonic(), lines.read(fault_pin))

        if settings["api"]["enabled"]:
            server = api.serve(settings["api"]["socket"], status, command)

        print("Betrieb gestartet")
        while not stopping:
            now = time.monotonic()
            controller.step(now)
            if screen:
                screen = draw(screen, controller, now)
            time.sleep(TICK)
    finally:
        try:
            controller.shutdown(time.monotonic())
        finally:
            if server:
                server.shutdown()
            if panel:
                panel.close()
            strip.close()
            lines.close()
    return 0


if __name__ == "__main__":
    sys.exit(main())
