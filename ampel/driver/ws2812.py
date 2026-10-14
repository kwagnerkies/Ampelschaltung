def spi_pattern(value):
    bits = 0
    for shift in range(7, -1, -1):
        bits = bits << 3 | (0b110 if value >> shift & 1 else 0b100)
    return bits.to_bytes(3, "big")


PATTERNS = tuple(spi_pattern(value) for value in range(256))


class Strip:
    PIXELS_PER_HEAD = 8
    COLORS = ((255, 0, 0), (255, 150, 0), (0, 255, 0))
    DARK = (0, 0, 0)
    RESET_BYTES = 60

    def __init__(self, bus, heads, brightness, pixels):
        self.bus = bus
        self.heads = heads
        self.brightness = brightness
        self.lamp_at = {pixel: lamp for lamp, pixel in enumerate(pixels)}

    def write(self, lamps):
        frame = bytearray()
        for head in range(self.heads):
            for pixel in range(self.PIXELS_PER_HEAD):
                lamp = self.lamp_at.get(pixel)
                lit = lamp is not None and lamps[head * 3 + lamp]
                red, green, blue = self.COLORS[lamp] if lit else self.DARK
                for value in (green, red, blue):
                    frame += PATTERNS[value * self.brightness // 255]
        frame += bytes(self.RESET_BYTES)
        self.bus.write(frame)

    def close(self):
        self.bus.close()
