import unittest

from .support import Lamps, aspects_of, run
from ..control import Controller, Timing
from ..phase import Stage
from ..rule import Following
from ..signal import Aspect, DIRECTIONS, check


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

    def test_starting_switched_off_stays_dark(self):
        self.controller.power(0.0, False)
        run(self.controller, 10)
        self.assertTrue(self.lamps.frames)
        for frame in self.lamps.frames:
            self.assertEqual(frame, [False] * 12)

    def test_warning_is_ignored_while_switched_off(self):
        now = run(self.controller, 8)
        self.controller.power(now, False)
        self.controller.warn(now, True)
        run(self.controller, 5, start=now)
        self.assertEqual(self.lamps.frames[-1], [False] * 12)

    def test_shutdown_after_power_off_in_green_stays_dark(self):
        now = run(self.controller, 8)
        self.assertIs(self.controller.machine.state.stage, Stage.GREEN)
        self.controller.power(now, False)
        self.controller.shutdown()
        self.assertEqual(self.lamps.frames[-1], [False] * 12)

    def test_shutdown_in_green_passes_yellow_to_red(self):
        run(self.controller, 8)
        self.controller.shutdown()
        self.assertIn(Aspect.YELLOW, aspects_of(self.lamps.frames[-2]))
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


if __name__ == "__main__":
    unittest.main()
