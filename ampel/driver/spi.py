import fcntl
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
