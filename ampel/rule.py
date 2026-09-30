class Following:
    def __init__(self, base, step, maximum):
        self.base = base
        self.step = step
        self.maximum = maximum

    def target(self, following):
        return min(self.base + self.step * following, self.maximum)

    def end_green(self, green, following):
        return green >= self.target(following)
