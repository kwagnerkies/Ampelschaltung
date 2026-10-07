import unittest



class ConfigTest(unittest.TestCase):
    def test_defaults_are_valid(self):
        from ..config import load
        self.assertEqual(load()["lamps"]["pixels"], [0, 4, 7])

    def test_pin_on_a_spi_line_is_refused(self):
        from ..config import ConfigError, validate, DEFAULTS
        settings = {section: dict(values) for section, values in DEFAULTS.items()}
        settings["sensors"] = dict(settings["sensors"], west=8)
        with self.assertRaises(ConfigError):
            validate(settings)

    def test_duplicate_pin_is_refused(self):
        from ..config import ConfigError, validate, DEFAULTS
        settings = {section: dict(values) for section, values in DEFAULTS.items()}
        settings["switches"] = dict(settings["switches"], fault=23)
        with self.assertRaises(ConfigError):
            validate(settings)

    def test_unknown_key_is_refused(self):
        import os
        import tempfile
        from ..config import ConfigError, load
        with tempfile.NamedTemporaryFile("w", suffix=".toml", delete=False) as handle:
            handle.write("[sensors]\nnort = 5\n")
        try:
            with self.assertRaises(ConfigError):
                load(handle.name)
        finally:
            os.remove(handle.name)

    def test_out_of_range_values_are_refused(self):
        from ..config import ConfigError, validate, DEFAULTS
        for section, key, value in (("lamps", "brightness", 300), ("timing", "yellow", 0),
                                    ("display", "rotation", "schraeg"), ("sensors", "north", "23")):
            settings = {name: dict(values) for name, values in DEFAULTS.items()}
            settings[section][key] = value
            with self.subTest(key=key), self.assertRaises(ConfigError):
                validate(settings)
