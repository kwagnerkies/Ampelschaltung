import unittest

from ampel.control import Controller, Timing
from ampel.phase import Stage
from ampel.rule import Following
from ampel.signal import Aspect, DIRECTIONS, ConflictError, check


class Lamps:
    def __init__(self):
        self.frames = []

    def write(self, state):
        self.frames.append(list(state))


def run(controller, seconds, step=0.05, start=0.0):
    now = start
    while now < start + seconds:
        controller.step(now)
        now += step
    return now


def aspects_of(frame):
    aspects = []
    for direction in DIRECTIONS:
        red, yellow, green = frame[direction * 3:direction * 3 + 3]
        if red and yellow:
            aspects.append(Aspect.RED_YELLOW)
        elif red:
            aspects.append(Aspect.RED)
        elif yellow:
            aspects.append(Aspect.YELLOW)
        elif green:
            aspects.append(Aspect.GREEN)
        else:
            aspects.append(Aspect.OFF)
    return aspects
