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
