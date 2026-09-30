import unittest

from ampel.signal import Aspect, ConflictError, check


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
