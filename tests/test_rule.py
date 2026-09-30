import unittest

from support import Lamps
from ampel.control import Controller, Timing
from ampel.phase import Stage
from ampel.rule import Following


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
