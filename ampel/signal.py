from enum import Enum


class Aspect(Enum):
    RED = "Rot"
    RED_YELLOW = "RotGelb"
    GREEN = "Gruen"
    YELLOW = "Gelb"
    OFF = "Aus"
    YELLOW_FLASH = "GelbBlinken"

    @property
    def lamps(self):
        return {
            Aspect.RED: (True, False, False),
            Aspect.RED_YELLOW: (True, True, False),
            Aspect.GREEN: (False, False, True),
            Aspect.YELLOW: (False, True, False),
            Aspect.YELLOW_FLASH: (False, True, False),
        }.get(self, (False, False, False))

    @property
    def releasing(self):
        return self in (Aspect.RED_YELLOW, Aspect.GREEN, Aspect.YELLOW)

    def can_follow(self, previous):
        if self is previous or self in (Aspect.YELLOW_FLASH, Aspect.OFF):
            return True
        return self in SUCCESSORS[previous]


SUCCESSORS = {
    Aspect.RED: (Aspect.RED_YELLOW,),
    Aspect.RED_YELLOW: (Aspect.GREEN, Aspect.RED),
    Aspect.GREEN: (Aspect.YELLOW,),
    Aspect.YELLOW: (Aspect.RED,),
    Aspect.OFF: (Aspect.RED,),
    Aspect.YELLOW_FLASH: (Aspect.RED,),
}

NORTH, EAST, SOUTH, WEST = range(4)
DIRECTIONS = (NORTH, EAST, SOUTH, WEST)
DIRECTION_NAMES = ("Nord", "Ost", "Sued", "West")

CONFLICTS = {
    NORTH: (EAST, WEST),
    EAST: (NORTH, SOUTH),
    SOUTH: (EAST, WEST),
    WEST: (NORTH, SOUTH),
}


class ConflictError(Exception):
    pass


def check(aspects):
    for direction, own in enumerate(aspects):
        if not own.releasing:
            continue
        for other in range(len(aspects)):
            if other == direction:
                continue
            if other in CONFLICTS[direction] and aspects[other].releasing:
                raise ConflictError(
                    f"{DIRECTION_NAMES[direction]} zeigt {own.value}, "
                    f"{DIRECTION_NAMES[other]} zeigt {aspects[other].value}"
                )
            if aspects[other] is Aspect.OFF:
                raise ConflictError(
                    f"{DIRECTION_NAMES[direction]} zeigt {own.value}, "
                    f"{DIRECTION_NAMES[other]} ist dunkel"
                )
