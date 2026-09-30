import unittest



class DriverTest(unittest.TestCase):
    def setUp(self):
        from ampel.driver import Strip

        class Tape:
            def __init__(self):
                self.frames = []

            def write(self, data):
                self.frames.append(bytes(data))

            def close(self):
                pass

        self.tape = Tape()
        self.strip = Strip(self.tape, 4, 255, (0, 4, 7))

    def decode(self, frame, pixel):
        values = []
        bit = pixel * 72
        for _ in range(3):
            value = 0
            for _ in range(8):
                pattern = 0
                for offset in range(3):
                    index = bit + offset
                    if frame[index // 8] & (0x80 >> (index % 8)):
                        pattern |= 1 << (2 - offset)
                value = (value << 1) | (1 if pattern == 0b110 else 0)
                bit += 3
            values.append(value)
        return values

    def test_each_head_lights_its_own_pixels(self):
        lamps = [False] * 12
        lamps[0] = lamps[4] = lamps[8] = True
        self.strip.write(lamps)
        frame = self.tape.frames[0]
        self.assertEqual(self.decode(frame, 0), [0, 255, 0])
        self.assertEqual(self.decode(frame, 8 + 4), [150, 255, 0])
        self.assertEqual(self.decode(frame, 16 + 7), [255, 0, 0])

    def test_dark_lamps_stay_black(self):
        self.strip.write([False] * 12)
        for pixel in range(32):
            self.assertEqual(self.decode(self.tape.frames[0], pixel), [0, 0, 0])

    def test_brightness_scales(self):
        from ampel.driver import Strip
        strip = Strip(self.tape, 4, 51, (0, 4, 7))
        lamps = [False] * 12
        lamps[0] = True
        strip.write(lamps)
        self.assertEqual(self.decode(self.tape.frames[0], 0)[1], 51)
