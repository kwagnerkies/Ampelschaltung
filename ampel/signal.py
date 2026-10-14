from enum import Enum


class Aspect(Enum):
    RED = "Rot"
    RED_YELLOW = "RotGelb"
    GREEN = "Gruen"
    YELLOW = "Gelb"
    OFF = "Aus"
    YELLOW_FLASH = "GelbBlinken"

    def lamps(self):
        if self is Aspect.RED:
            return [True, False, False]
        if self is Aspect.RED_YELLOW:
            return [True, True, False]
        if self is Aspect.GREEN:
            return [False, False, True]
        if self is Aspect.YELLOW or self is Aspect.YELLOW_FLASH:
            return [False, True, False]
        return [False, False, False]

    def is_releasing(self):
        return self in (Aspect.RED_YELLOW, Aspect.GREEN, Aspect.YELLOW)

    def can_follow(self, previous):
        if self is previous or self in (Aspect.YELLOW_FLASH, Aspect.OFF):
            return True
        return self in SUCCESSORS[previous]


SUCCESSORS = {
    Aspect.RED: [Aspect.RED_YELLOW],
    Aspect.RED_YELLOW: [Aspect.GREEN, Aspect.RED],
    Aspect.GREEN: [Aspect.YELLOW],
    Aspect.YELLOW: [Aspect.RED],
    Aspect.OFF: [Aspect.RED],
    Aspect.YELLOW_FLASH: [Aspect.RED],
}

NORTH = 0
EAST = 1
SOUTH = 2
WEST = 3
DIRECTIONS = [NORTH, EAST, SOUTH, WEST]
DIRECTION_NAMES = ["Nord", "Ost", "Sued", "West"]

CONFLICTS = {
    NORTH: [EAST, WEST],
    EAST: [NORTH, SOUTH],
    SOUTH: [EAST, WEST],
    WEST: [NORTH, SOUTH],
}


class ConflictError(Exception):
    pass


def check(aspects):
    for direction in DIRECTIONS:
        own = aspects[direction]
        if not own.is_releasing():
            continue
        for other in DIRECTIONS:
            if other in CONFLICTS[direction] and aspects[other].is_releasing():
                raise ConflictError(f"{DIRECTION_NAMES[direction]} zeigt {own.value}, "
                                    f"{DIRECTION_NAMES[other]} zeigt {aspects[other].value}")
            if aspects[other] is Aspect.OFF:
                raise ConflictError(f"{DIRECTION_NAMES[direction]} zeigt {own.value}, "
                                    f"{DIRECTION_NAMES[other]} ist dunkel")
