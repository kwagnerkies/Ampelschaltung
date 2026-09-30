import unittest

from ampel.control import Controller, Timing
from ampel.phase import Phase, Stage
from ampel.rule import Following
from ampel.signal import Aspect, DIRECTIONS, ConflictError, NORTH, EAST, check


class Lamps:
    def __init__(self):
        self.frames = []

    def write(self, state):
        self.frames.append(list(state))


def run(controller, seconds, step=0.05, start=0.0):
    now = start
    while now < start + seconds:
        controller.step(now)
        now += step
    return now


class SafetyTest(unittest.TestCase):
    def test_crossing_greens_are_refused(self):
        with self.assertRaises(ConflictError):
            check([Aspect.GREEN, Aspect.GREEN, Aspect.RED, Aspect.RED])

    def test_opposite_greens_are_fine(self):
        check([Aspect.GREEN, Aspect.RED, Aspect.GREEN, Aspect.RED])

    def test_dark_head_next_to_a_release_is_refused(self):
        with self.assertRaises(ConflictError):
            check([Aspect.GREEN, Aspect.OFF, Aspect.GREEN, Aspect.RED])

    def test_red_may_not_follow_green_directly(self):
        self.assertFalse(Aspect.RED.can_follow(Aspect.GREEN))
        self.assertTrue(Aspect.YELLOW.can_follow(Aspect.GREEN))


class SequenceTest(unittest.TestCase):
    def setUp(self):
        self.lamps = Lamps()
        self.controller = Controller(self.lamps, Following(5, 3, 20), Timing())

    def test_phases_cycle_in_order(self):
        seen = []
        now = 0.0
        while now < 60:
            self.controller.step(now)
            name = self.controller.machine.state.name
            if not seen or seen[-1] != name:
                seen.append(name)
            now += 0.05
        self.assertEqual(
            seen[:9],
            ["Allrot", "OW_RotGelb", "OW_Gruen", "OW_Gelb", "Allrot",
             "NS_RotGelb", "NS_Gruen", "NS_Gelb", "Allrot"],
        )

    def test_every_written_frame_is_legal(self):
        run(self.controller, 120)
        previous = None
        for frame in self.lamps.frames:
            aspects = aspects_of(frame)
            check(aspects)
            if previous:
                for direction in DIRECTIONS:
                    self.assertTrue(aspects[direction].can_follow(previous[direction]))
            previous = aspects


class RuleTest(unittest.TestCase):
    def setUp(self):
        self.controller = Controller(Lamps(), Following(5, 3, 20), Timing())

    def green_phase(self):
        now = 0.0
        while self.controller.machine.state.stage is not Stage.GREEN:
            self.controller.step(now)
            now += 0.05
        return now

    def test_two_close_vehicles_extend(self):
        now = self.green_phase()
        direction = self.controller.machine.state.phase.directions[0]
        self.controller.crossing(now, direction)
        self.controller.crossing(now + 0.7, direction)
        self.assertEqual(self.controller.following, 1)
        self.assertEqual(self.controller.rule.target(self.controller.following), 8)

    def test_scattered_vehicles_do_not_extend(self):
        now = self.green_phase()
        direction = self.controller.machine.state.phase.directions[0]
        self.controller.crossing(now, direction)
        self.controller.crossing(now + 5, direction)
        self.assertEqual(self.controller.following, 0)

    def test_opposite_approach_is_not_a_sequence(self):
        now = self.green_phase()
        first, second = self.controller.machine.state.phase.directions
        self.controller.crossing(now, first)
        self.controller.crossing(now + 0.1, second)
        self.assertEqual(self.controller.following, 0)

    def test_extension_stops_at_the_maximum(self):
        self.assertEqual(self.controller.rule.target(99), 20)


class SwitchTest(unittest.TestCase):
    def setUp(self):
        self.lamps = Lamps()
        self.controller = Controller(self.lamps, Following(5, 3, 20), Timing())

    def test_power_off_darkens_everything(self):
        now = run(self.controller, 10)
        self.controller.power(now, False)
        self.assertEqual(self.lamps.frames[-1], [False] * 12)
        frames = len(self.lamps.frames)
        run(self.controller, 60, start=now)
        self.assertEqual(len(self.lamps.frames), frames)

    def test_power_on_starts_at_all_red(self):
        now = run(self.controller, 10)
        self.controller.power(now, False)
        self.controller.power(now + 1, True)
        self.assertEqual(aspects_of(self.lamps.frames[-1]), [Aspect.RED] * 4)

    def test_warning_flashes_yellow_only(self):
        now = run(self.controller, 8)
        self.controller.warn(now, True)
        run(self.controller, 10, start=now)
        for frame in self.lamps.frames[-10:]:
            for direction in DIRECTIONS:
                red, yellow, green = frame[direction * 3:direction * 3 + 3]
                self.assertFalse(red)
                self.assertFalse(green)

    def test_warning_off_restarts_at_all_red(self):
        now = run(self.controller, 8)
        self.controller.warn(now, True)
        now = run(self.controller, 5, start=now)
        self.controller.warn(now, False)
        self.assertEqual(aspects_of(self.lamps.frames[-1]), [Aspect.RED] * 4)


class DisplayValuesTest(unittest.TestCase):
    def test_running_direction_counts_down_and_jumps(self):
        controller = Controller(Lamps(), Following(5, 3, 20), Timing())
        now = 0.0
        while controller.machine.state.stage is not Stage.GREEN:
            controller.step(now)
            now += 0.05
        controller.step(now + 1)
        direction = controller.machine.state.phase.directions[0]
        before = controller.snapshot(now + 1).green[direction]
        controller.crossing(now + 1, direction)
        controller.crossing(now + 1.5, direction)
        controller.step(now + 1.6)
        after = controller.snapshot(now + 1.6).green[direction]
        self.assertGreater(after, before)


def aspects_of(frame):
    aspects = []
    for direction in DIRECTIONS:
        red, yellow, green = frame[direction * 3:direction * 3 + 3]
        if red and yellow:
            aspects.append(Aspect.RED_YELLOW)
        elif red:
            aspects.append(Aspect.RED)
        elif yellow:
            aspects.append(Aspect.YELLOW)
        elif green:
            aspects.append(Aspect.GREEN)
        else:
            aspects.append(Aspect.OFF)
    return aspects


if __name__ == "__main__":
    unittest.main()


class DriverTest(unittest.TestCase):
    def setUp(self):
        from ampel.driver import Strip

        class Tape:
            def __init__(self):
                self.frames = []

            def write(self, data):
                self.frames.append(bytes(data))

            def close(self):
                pass

        self.tape = Tape()
        self.strip = Strip(self.tape, 4, 255, (0, 4, 7))

    def decode(self, frame, pixel):
        values = []
        bit = pixel * 72
        for _ in range(3):
            value = 0
            for _ in range(8):
                pattern = 0
                for offset in range(3):
                    index = bit + offset
                    if frame[index // 8] & (0x80 >> (index % 8)):
                        pattern |= 1 << (2 - offset)
                value = (value << 1) | (1 if pattern == 0b110 else 0)
                bit += 3
            values.append(value)
        return values

    def test_each_head_lights_its_own_pixels(self):
        lamps = [False] * 12
        lamps[0] = lamps[4] = lamps[8] = True
        self.strip.write(lamps)
        frame = self.tape.frames[0]
        self.assertEqual(self.decode(frame, 0), [0, 255, 0])
        self.assertEqual(self.decode(frame, 8 + 4), [150, 255, 0])
        self.assertEqual(self.decode(frame, 16 + 7), [255, 0, 0])

    def test_dark_lamps_stay_black(self):
        self.strip.write([False] * 12)
        for pixel in range(32):
            self.assertEqual(self.decode(self.tape.frames[0], pixel), [0, 0, 0])

    def test_brightness_scales(self):
        from ampel.driver import Strip
        strip = Strip(self.tape, 4, 51, (0, 4, 7))
        lamps = [False] * 12
        lamps[0] = True
        strip.write(lamps)
        self.assertEqual(self.decode(self.tape.frames[0], 0)[1], 51)


class ScreenTest(unittest.TestCase):
    def setUp(self):
        from ampel.display import Screen

        class Canvas:
            def __init__(self):
                self.fills = []

            def size(self):
                return (320, 240)

            def fill(self, x, y, width, height, color):
                self.fills.append((x, y, width, height, color))

        self.canvas = Canvas()
        self.screen = Screen(self.canvas)

    def test_first_update_draws_everything(self):
        from ampel.display import GREEN, RED
        self.screen.update({0: (18, GREEN), 1: (5, RED), 2: (18, GREEN), 3: (5, RED)})
        self.assertEqual(len(self.canvas.fills), 1 + 4 * 2 * 7)

    def test_only_changed_fields_are_redrawn(self):
        from ampel.display import GREEN, RED
        self.screen.update({0: (18, GREEN), 1: (5, RED), 2: (18, GREEN), 3: (5, RED)})
        self.canvas.fills.clear()
        self.screen.update({0: (18, GREEN), 1: (12, RED), 2: (18, GREEN), 3: (12, RED)})
        self.assertEqual(len(self.canvas.fills), 2 * 2 * 7)

    def test_cross_layout(self):
        north, east, south, west = (self.screen.boxes[d] for d in range(4))
        self.assertLess(north[1], south[1])
        self.assertLess(west[0], east[0])


class ConfigTest(unittest.TestCase):
    def test_defaults_are_valid(self):
        from ampel.config import load
        self.assertEqual(load()["lamps"]["pixels"], [0, 4, 7])

    def test_pin_on_a_spi_line_is_refused(self):
        from ampel.config import ConfigError, validate, DEFAULTS
        settings = {section: dict(values) for section, values in DEFAULTS.items()}
        settings["sensors"] = dict(settings["sensors"], west=8)
        with self.assertRaises(ConfigError):
            validate(settings)

    def test_duplicate_pin_is_refused(self):
        from ampel.config import ConfigError, validate, DEFAULTS
        settings = {section: dict(values) for section, values in DEFAULTS.items()}
        settings["switches"] = dict(settings["switches"], fault=23)
        with self.assertRaises(ConfigError):
            validate(settings)


class ApiTest(unittest.TestCase):
    def test_status_reports_the_snapshot(self):
        from ampel.api import status_of
        controller = Controller(Lamps(), Following(5, 3, 20), Timing())
        controller.step(0.0)
        controller.step(10.0)
        status = status_of(controller.snapshot(10.0))
        self.assertTrue(status["an"])
        self.assertFalse(status["notzustand"])
        self.assertIn("Nord", status["gruenzeiten_s"])
