from .signal import Aspect, DIRECTIONS

BLACK, GREY, RED, YELLOW, GREEN, WHITE = 0x0000, 0x39E7, 0xF800, 0xFFE0, 0x07E0, 0xFFFF

SEGMENTS = (
    (1, 1, 1, 1, 1, 1, 0),
    (0, 1, 1, 0, 0, 0, 0),
    (1, 1, 0, 1, 1, 0, 1),
    (1, 1, 1, 1, 0, 0, 1),
    (0, 1, 1, 0, 0, 1, 1),
    (1, 0, 1, 1, 0, 1, 1),
    (1, 0, 1, 1, 1, 1, 1),
    (1, 1, 1, 0, 0, 0, 0),
    (1, 1, 1, 1, 1, 1, 1),
    (1, 1, 1, 1, 0, 1, 1),
)

COLORS = {
    Aspect.GREEN: GREEN,
    Aspect.YELLOW: YELLOW,
    Aspect.RED_YELLOW: YELLOW,
    Aspect.YELLOW_FLASH: YELLOW,
    Aspect.RED: RED,
}


class Screen:
    def __init__(self, canvas):
        self.canvas = canvas
        width, height = canvas.size()
        box_width, box_height = width // 3, height // 4
        center_x, center_y = (width - box_width) // 2, (height - box_height) // 2
        margin = box_height // 4
        self.boxes = {
            0: (center_x, margin, box_width, box_height),
            1: (width - box_width - margin, center_y, box_width, box_height),
            2: (center_x, height - box_height - margin, box_width, box_height),
            3: (margin, center_y, box_width, box_height),
        }
        self.last = {}

    def update(self, fields):
        if not self.last:
            width, height = self.canvas.size()
            self.canvas.fill(0, 0, width, height, BLACK)
        for direction in DIRECTIONS:
            if self.last.get(direction) == fields[direction]:
                continue
            seconds, color = fields[direction]
            self._number(self.boxes[direction], seconds, color)
            self.last[direction] = fields[direction]

    def clear(self):
        width, height = self.canvas.size()
        self.canvas.fill(0, 0, width, height, BLACK)
        self.last = {}

    def _number(self, box, value, color):
        x, y, width, height = box
        value = max(0, min(99, value))
        gap = width // 10
        digit = (width - gap) // 2
        self._digit(x, y, digit, height, value // 10, color)
        self._digit(x + digit + gap, y, digit, height, value % 10, color)

    def _digit(self, x, y, width, height, value, color):
        thick = max(2, width // 5)
        half = (height - thick) // 2
        boxes = (
            (x, y, width, thick),
            (x + width - thick, y, thick, half),
            (x + width - thick, y + half, thick, half + thick),
            (x, y + height - thick, width, thick),
            (x, y + half, thick, half + thick),
            (x, y, thick, half),
            (x, y + half, width, thick),
        )
        for lit, box in zip(SEGMENTS[value], boxes):
            self.canvas.fill(*box, color if lit else GREY)


def fields_from(snapshot):
    return {
        direction: (
            round(snapshot.green[direction]),
            COLORS.get(snapshot.aspects[direction], GREY),
        )
        for direction in DIRECTIONS
    }
