class Strip:
    PIXELS_PER_HEAD = 8
    COLORS = ((255, 0, 0), (255, 150, 0), (0, 255, 0))
    RESET_BYTES = 60

    def __init__(self, bus, heads, brightness, pixels):
        self.bus = bus
        self.heads = heads
        self.brightness = brightness
        self.pixels = pixels
        self.frame = bytearray(heads * self.PIXELS_PER_HEAD * 9 + self.RESET_BYTES)

    def write(self, lamps):
        for i in range(len(self.frame)):
            self.frame[i] = 0
        for head in range(self.heads):
            for lamp, pixel in enumerate(self.pixels):
                if lamps[head * 3 + lamp]:
                    self._encode(head * self.PIXELS_PER_HEAD + pixel, self.COLORS[lamp])
        self.bus.write(self.frame)

    def close(self):
        self.bus.close()

    def _encode(self, pixel, color):
        red, green, blue = color
        bit = pixel * 72
        for value in (green, red, blue):
            value = value * self.brightness // 255
            for shift in range(7, -1, -1):
                pattern = 0b110 if value >> shift & 1 else 0b100
                for offset in range(3):
                    if pattern >> (2 - offset) & 1:
                        index = bit + offset
                        self.frame[index // 8] |= 0x80 >> (index % 8)
                bit += 3
