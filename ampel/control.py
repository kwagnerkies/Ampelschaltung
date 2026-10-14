import sys
import threading
import time

from .phase import Machine, Phase, Stage, phase_of
from .signal import Aspect, ConflictError, DIRECTIONS, check

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
        for direction in DIRECTIONS:
            previous = self.shown[direction]
            if not aspects[direction].can_follow(previous):
                raise ConflictError(f"{aspects[direction].value} darf nicht auf "
                                    f"{previous.value} folgen")
        state = []
        for aspect in aspects:
            state.extend(aspect.lamps())
        self.lamps.write(state)
        self.shown = list(aspects)

    def show_all(self, aspect):
        self.show([aspect for _ in DIRECTIONS])


class Controller:
    def __init__(self, lamps, rule, timing=None, follow=2.0, now=0.0):
        if timing is None:
            timing = Timing()
        self.output = Output(lamps)
        self.rule = rule
        self.timing = timing
        self.follow = follow
        self.machine = Machine(timing, now)
        self.following = 0
        self.last_crossing = [None for _ in DIRECTIONS]
        self.flash_on = False
        self.warning = False
        self.on = True
        self.begun = False
        self.lock = threading.RLock()

    def step(self, now):
        with self.lock:
            if not self.begun:
                self.begun = True
                self.output.show(self.machine.state.aspects())
            if not self.on:
                return
            if self.is_flashing():
                self._flash(now)
                return
            self._run_phases(now)

    def _run_phases(self, now):
        state = self.machine.state
        end_green = False
        if state.stage is Stage.GREEN:
            end_green = self.rule.end_green(now - state.since, self.following)
        if self.machine.advance(now, end_green):
            state = self.machine.state
            if state.stage is Stage.GREEN:
                self.following = 0
                self.last_crossing = [None for _ in DIRECTIONS]
            try:
                self.output.show(state.aspects())
            except Exception as error:
                self.enter_fault(now, error)
                return
        if state.stage is Stage.GREEN:
            state.target = self.rule.target(self.following)

    def crossing(self, now, direction):
        with self.lock:
            state = self.machine.state
            if not self.on or state.stage is not Stage.GREEN:
                return
            if phase_of(direction) is not state.phase:
                return
            last = self.last_crossing[direction]
            if last is not None and now - last <= self.follow:
                self.following += 1
            self.last_crossing[direction] = now

    def power(self, now, on):
        with self.lock:
            if on == self.on and self.begun:
                return
            self.on = on
            self.begun = True
            if not on:
                self.output.show_all(Aspect.OFF)
                return
            self.restart(now)
            if self.warning:
                self.enter_fault(now, "Notschalter geschlossen")

    def warn(self, now, on):
        with self.lock:
            if on == self.warning:
                return
            self.warning = on
            if not self.on:
                return
            if on:
                self.enter_fault(now, "Notschalter geschlossen")
            else:
                self.restart(now)

    def restart(self, now):
        self.flash_on = False
        self.following = 0
        self.last_crossing = [None for _ in DIRECTIONS]
        self.machine.restart(now)
        self.output.show(self.machine.state.aspects())

    def enter_fault(self, now, reason):
        print(f"Notzustand: {reason}", file=sys.stderr)
        self.machine.fault(now)
        self.flash_on = False
        self._flash(now)

    def shutdown(self, now, wait=time.sleep):
        with self.lock:
            if not self.on or not self.begun:
                return
            state = self.machine.state
            if state.stage is Stage.YELLOW:
                remaining = self.timing.yellow - (now - state.since)
                if remaining > 0:
                    wait(remaining)
            elif Aspect.GREEN in self.output.shown:
                yellow = []
                for aspect in self.output.shown:
                    if aspect is Aspect.GREEN:
                        yellow.append(Aspect.YELLOW)
                    else:
                        yellow.append(aspect)
                self.output.show(yellow)
                wait(self.timing.yellow)
            self.output.show_all(Aspect.RED)

    def is_flashing(self):
        return self.on and (self.warning or self.machine.state.phase is Phase.FAULT)

    def snapshot(self, now):
        with self.lock:
            state = self.machine.state
            green = {}
            for direction in DIRECTIONS:
                if phase_of(direction) is state.phase and state.stage is Stage.GREEN:
                    green[direction] = max(0.0, state.target - (now - state.since))
                else:
                    green[direction] = self.rule.target(0)
            if self.is_flashing():
                aspects = [Aspect.YELLOW_FLASH for _ in DIRECTIONS]
            else:
                aspects = list(self.output.shown)
            warning = self.warning or self.is_flashing()
            return Snapshot(state, self.following, aspects, green, self.on, warning)

    def _flash(self, now):
        half_periods = int((now - self.machine.state.since) / FLASH_HALF)
        lit = half_periods % 2 == 0
        if lit == self.flash_on:
            return
        self.flash_on = lit
        try:
            if lit:
                self.output.show_all(Aspect.YELLOW_FLASH)
            else:
                self.output.show_all(Aspect.OFF)
        except Exception as error:
            print(f"Lampen nicht erreichbar: {error}", file=sys.stderr)


class Snapshot:
    def __init__(self, state, following, aspects, green, on, warning):
        self.state = state
        self.following = following
        self.aspects = aspects
        self.green = green
        self.on = on
        self.warning = warning
