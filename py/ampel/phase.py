from enum import Enum

from .signal import Aspect, DIRECTIONS, EAST, NORTH, SOUTH, WEST


class Phase(Enum):
    STARTUP = "Start"
    NS = "NS"
    EW = "OW"
    FAULT = "Stoerung"

    @property
    def directions(self):
        return {Phase.NS: (NORTH, SOUTH), Phase.EW: (EAST, WEST)}.get(self, ())

    @property
    def other(self):
        return Phase.NS if self is Phase.EW else Phase.EW


class Stage(Enum):
    GREEN = "Gruen"
    YELLOW = "Gelb"
    ALL_RED = "Allrot"
    RED_YELLOW = "RotGelb"


def phase_of(direction):
    return Phase.NS if direction in (NORTH, SOUTH) else Phase.EW


class State:
    def __init__(self, phase, stage, since, target=0.0):
        self.phase = phase
        self.stage = stage
        self.since = since
        self.target = target

    @property
    def name(self):
        if self.stage is Stage.ALL_RED:
            return Stage.ALL_RED.value
        return f"{self.phase.value}_{self.stage.value}"

    @property
    def aspects(self):
        showing = {
            Stage.GREEN: Aspect.GREEN,
            Stage.YELLOW: Aspect.YELLOW,
            Stage.RED_YELLOW: Aspect.RED_YELLOW,
        }.get(self.stage)
        aspects = [Aspect.RED for _ in DIRECTIONS]
        if showing:
            for direction in self.phase.directions:
                aspects[direction] = showing
        return aspects


class Machine:
    def __init__(self, timing, now):
        self.timing = timing
        self.state = State(Phase.STARTUP, Stage.ALL_RED, now)

    def advance(self, now, end_green):
        elapsed = now - self.state.since
        stage = self.state.stage
        if stage is Stage.GREEN:
            if not end_green:
                return False
            self._enter(now, self.state.phase, Stage.YELLOW)
        elif stage is Stage.YELLOW:
            if elapsed < self.timing.yellow:
                return False
            self._enter(now, self.state.phase, Stage.ALL_RED)
        elif stage is Stage.ALL_RED:
            if elapsed < self.timing.all_red:
                return False
            self._enter(now, self.state.phase.other, Stage.RED_YELLOW)
        else:
            if elapsed < self.timing.red_yellow:
                return False
            self._enter(now, self.state.phase, Stage.GREEN)
        return True

    def restart(self, now):
        self.state = State(Phase.STARTUP, Stage.ALL_RED, now)

    def fault(self, now):
        self.state = State(Phase.FAULT, Stage.ALL_RED, now)

    def _enter(self, now, phase, stage):
        target = self.state.target if stage is Stage.GREEN else 0.0
        self.state = State(phase, stage, now, target)
