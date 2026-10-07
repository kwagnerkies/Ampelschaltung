import unittest



class ScreenTest(unittest.TestCase):
    def setUp(self):
        from ..display import Screen

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
        from ..display import GREEN, RED
        self.screen.update({0: (18, GREEN), 1: (5, RED), 2: (18, GREEN), 3: (5, RED)})
        self.assertEqual(len(self.canvas.fills), 1 + 4 * 2 * 7)

    def test_only_changed_fields_are_redrawn(self):
        from ..display import GREEN, RED
        self.screen.update({0: (18, GREEN), 1: (5, RED), 2: (18, GREEN), 3: (5, RED)})
        self.canvas.fills.clear()
        self.screen.update({0: (18, GREEN), 1: (12, RED), 2: (18, GREEN), 3: (12, RED)})
        self.assertEqual(len(self.canvas.fills), 2 * 2 * 7)

    def test_cross_layout(self):
        north, east, south, west = (self.screen.boxes[d] for d in range(4))
        self.assertLess(north[1], south[1])
        self.assertLess(west[0], east[0])

    def test_lit_segments_are_drawn_last(self):
        from ..display import GREEN, GREY
        self.screen.update({0: (11, GREEN), 1: (11, GREEN), 2: (11, GREEN), 3: (11, GREEN)})
        digit = [fill[4] for fill in self.canvas.fills[1:8]]
        self.assertEqual(digit, [GREY] * 5 + [GREEN] * 2)
