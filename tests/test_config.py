import unittest



class ConfigTest(unittest.TestCase):
    def test_defaults_are_valid(self):
        from ampel.config import load
        self.assertEqual(load()["lamps"]["pixels"], [0, 4, 7])

    def test_pin_on_a_spi_line_is_refused(self):
        from ampel.config import ConfigError, validate, DEFAULTS
        settings = {section: dict(values) for section, values in DEFAULTS.items()}
        settings["sensors"] = dict(settings["sensors"], west=8)
        with self.assertRaises(ConfigError):
            validate(settings)

    def test_duplicate_pin_is_refused(self):
        from ampel.config import ConfigError, validate, DEFAULTS
        settings = {section: dict(values) for section, values in DEFAULTS.items()}
        settings["switches"] = dict(settings["switches"], fault=23)
        with self.assertRaises(ConfigError):
            validate(settings)
