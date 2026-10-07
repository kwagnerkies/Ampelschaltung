import threading

from .phase import Machine, Phase, Stage, phase_of
from .signal import Aspect, DIRECTIONS, check

FLASH_HALF = 0.5


class Timing:
    def __init__(self, yellow=3.0, all_red=2.0, red_yellow=1.0):
        self.yellow = yellow
        self.all_red = all_red
        self.red_yellow = red_yellow


class Output:
    def __init__(self, lamps):
        self.lamps = lamps
        self.shown = [Aspect.OFF for _ in DIRECTIONS]

    def show(self, aspects):
        check(aspects)
        for direction, aspect in enumerate(aspects):
            if not aspect.can_follow(self.shown[direction]):
                raise ConflictOrder(direction, self.shown[direction], aspect)
        state = []
        for aspect in aspects:
            state.extend(aspect.lamps)
        self.lamps.write(state)
        self.shown = list(aspects)

    def show_all(self, aspect):
        self.show([aspect for _ in DIRECTIONS])


class ConflictOrder(Exception):
    def __init__(self, direction, previous, aspect):
        super().__init__(f"{aspect.value} darf nicht auf {previous.value} folgen")


class Controller:
    def __init__(self, lamps, rule, timing=None, follow=2.0, now=0.0, observers=()):
        self.output = Output(lamps)
        self.rule = rule
        self.timing = timing or Timing()
        self.follow = follow
        self.machine = Machine(self.timing, now)
        self.observers = list(observers)
        self.following = 0
        self.last_crossing = [None] * len(DIRECTIONS)
        self.flash_on = False
        self.warning = False
        self.on = True
        self.begun = False
        self.lock = threading.RLock()

    def step(self, now):
        with self.lock:
            self._step(now)

    def _step(self, now):
        if not self.begun:
            self.begun = True
            if self.on:
                self.output.show(self.machine.state.aspects)
                self._notify("phase", now, self.machine.state)
            else:
                self.output.show_all(Aspect.OFF)
        if not self.on:
            return
        if self.warning or self.machine.state.phase is Phase.FAULT:
            self._flash(now)
            return

        state = self.machine.state
        end = state.stage is Stage.GREEN and self.rule.end_green(now - state.since, self.following)
        if self.machine.advance(now, end):
            state = self.machine.state
            if state.stage is Stage.GREEN:
                self.following = 0
                self.last_crossing = [None] * len(DIRECTIONS)
            try:
                self.output.show(state.aspects)
            except Exception as error:
                self.enter_fault(now, error)
                return
            self._notify("phase", now, state)
        if self.machine.state.stage is Stage.GREEN:
            self.machine.state.target = self.rule.target(self.following)

    def crossing(self, now, direction):
        with self.lock:
            state = self.machine.state
            if not self.on or state.stage is not Stage.GREEN or phase_of(direction) is not state.phase:
                return
            last = self.last_crossing[direction]
            if last is not None and now - last <= self.follow:
                self.following += 1
            self.last_crossing[direction] = now
            self._notify("sensor", now, direction)

    def power(self, now, on):
        with self.lock:
            if on == self.on and self.begun:
                return
            self.on = on
            self.begun = True
            if on:
                self.restart(now)
            else:
                self.warning = False
                self.output.show_all(Aspect.OFF)
                self._notify("power", now, False)

    def warn(self, now, on):
        with self.lock:
            if not self.on or on == self.warning:
                return
            self.warning = on
            if on:
                self.enter_fault(now, Warning("notzustand ueber den schalter"))
            else:
                self.restart(now)

    def restart(self, now):
        with self.lock:
            self.warning = False
            self.flash_on = False
            self.following = 0
            self.last_crossing = [None] * len(DIRECTIONS)
            self.machine.restart(now)
            self.output.show(self.machine.state.aspects)
            self._notify("power", now, True)
            self._notify("phase", now, self.machine.state)

    def enter_fault(self, now, error):
        with self.lock:
            self.machine.fault(now)
            self.flash_on = False
            self._notify("fault", now, error)
            self._flash(now)

    def shutdown(self):
        with self.lock:
            if not self.on:
                return
            shown = self.output.shown
            if Aspect.GREEN in shown:
                self.output.show([Aspect.YELLOW if aspect is Aspect.GREEN else aspect
                                  for aspect in shown])
            self.output.show_all(Aspect.RED)

    def snapshot(self, now):
        with self.lock:
            state = self.machine.state
            green = {}
            for direction in DIRECTIONS:
                if phase_of(direction) is state.phase and state.stage is Stage.GREEN:
                    green[direction] = max(0.0, state.target - (now - state.since))
                else:
                    green[direction] = self.rule.target(0)
            return Snapshot(state, self.following, list(self.output.shown), green,
                            self.on, self.warning)

    def _flash(self, now):
        on = int((now - self.machine.state.since) / FLASH_HALF) % 2 == 0
        if on == self.flash_on:
            return
        self.flash_on = on
        self.output.show_all(Aspect.YELLOW_FLASH if on else Aspect.OFF)

    def _notify(self, event, now, payload):
        for observer in self.observers:
            observer(event, now, payload)


class Snapshot:
    def __init__(self, state, following, aspects, green, on, warning):
        self.state = state
        self.following = following
        self.aspects = aspects
        self.green = green
        self.on = on
        self.warning = warning
