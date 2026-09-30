import fcntl
import time
import os
import struct

SPI_MODE = 0x40016B01
SPI_BITS = 0x40016B03
SPI_SPEED = 0x40046B04


class Spi:
    def __init__(self, device, speed_hz):
        self.file = os.open(device, os.O_RDWR)
        for request, value in ((SPI_MODE, 0), (SPI_BITS, 8), (SPI_SPEED, speed_hz)):
            fcntl.ioctl(self.file, request, struct.pack("I", value))

    def write(self, data):
        os.write(self.file, bytes(data))

    def close(self):
        os.close(self.file)


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


class Tft:
    LANDSCAPE = 0x28
    PORTRAIT = 0x48
    INIT = (
        (0x01, b"", 0.15),
        (0xCF, b"\x00\xC1\x30", 0),
        (0xED, b"\x64\x03\x12\x81", 0),
        (0xE8, b"\x85\x00\x78", 0),
        (0xCB, b"\x39\x2C\x00\x34\x02", 0),
        (0xF7, b"\x20", 0),
        (0xEA, b"\x00\x00", 0),
        (0xC0, b"\x23", 0),
        (0xC1, b"\x10", 0),
        (0xC5, b"\x3E\x28", 0),
        (0xC7, b"\x86", 0),
        (0x3A, b"\x55", 0),
        (0xB1, b"\x00\x18", 0),
        (0xB6, b"\x08\x82\x27", 0),
        (0xF2, b"\x00", 0),
        (0x26, b"\x01", 0),
        (0x11, b"", 0.12),
        (0x29, b"", 0.02),
    )

    def __init__(self, bus, dc, rotation=LANDSCAPE):
        self.bus = bus
        self.dc = dc
        self.width, self.height = (320, 240) if rotation == self.LANDSCAPE else (240, 320)
        for command, data, pause in self.INIT:
            self._command(command, data)
            if pause:
                time.sleep(pause)
        self._command(0x36, bytes([rotation]))

    def size(self):
        return self.width, self.height

    def fill(self, x, y, width, height, color):
        if width <= 0 or height <= 0:
            return
        self._command(0x2A, struct.pack(">HH", x, x + width - 1))
        self._command(0x2B, struct.pack(">HH", y, y + height - 1))
        self._command(0x2C)
        self.dc.set(True)
        chunk = struct.pack(">H", color) * 1024
        remaining = width * height
        while remaining > 0:
            count = min(remaining, 1024)
            self.bus.write(chunk[: count * 2])
            remaining -= count

    def close(self):
        self.bus.close()

    def _command(self, code, data=b""):
        self.dc.set(False)
        self.bus.write(bytes([code]))
        if data:
            self.dc.set(True)
            self.bus.write(data)
