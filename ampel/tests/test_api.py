import unittest

from .support import Lamps
from ..control import Controller, Timing
from ..rule import Following


class ApiTest(unittest.TestCase):
    def test_status_reports_the_snapshot(self):
        from ..api import status_of
        controller = Controller(Lamps(), Following(5, 3, 20), Timing())
        controller.step(0.0)
        controller.step(10.0)
        status = status_of(controller.snapshot(10.0))
        self.assertTrue(status["an"])
        self.assertFalse(status["notzustand"])
        self.assertIn("Nord", status["gruenzeiten_s"])
