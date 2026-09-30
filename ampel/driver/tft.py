import struct
import time


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
