from gpiozero import Button, DigitalOutputDevice


class Lines:
    def __init__(self):
        self.devices = []

    def output(self, pin):
        device = DigitalOutputDevice(pin)
        self.devices.append(device)
        return Output(device)

    def watch(self, pins, handler):
        for pin in pins:
            button = Button(pin, pull_up=True, bounce_time=0.015)
            button.when_pressed = lambda pin=pin: handler(pin, True)
            button.when_released = lambda pin=pin: handler(pin, False)
            self.devices.append(button)

    def read(self, pin):
        for device in self.devices:
            if isinstance(device, Button) and device.pin.number == pin:
                return device.is_pressed
        return False

    def close(self):
        for device in self.devices:
            device.close()


class Output:
    def __init__(self, device):
        self.device = device

    def set(self, high):
        self.device.value = 1 if high else 0
