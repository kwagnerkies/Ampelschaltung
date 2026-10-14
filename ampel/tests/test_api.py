import contextlib
import io
import json
import os
import socket
import unittest

from .support import Lamps
from ..control import Controller, Timing
from ..rule import Following


class ApiTest(unittest.TestCase):
    def setUp(self):
        self.enterContext(contextlib.redirect_stderr(io.StringIO()))

    def test_status_reports_the_snapshot(self):
        from ..api import status_of
        controller = Controller(Lamps(), Following(5, 3, 20), Timing())
        controller.step(0.0)
        controller.step(10.0)
        status = status_of(controller.snapshot(10.0))
        self.assertTrue(status["an"])
        self.assertFalse(status["notzustand"])
        self.assertIn("Nord", status["gruenzeiten_s"])

    def test_warning_reports_fault_and_lit_lamps(self):
        from ..api import status_of
        controller = Controller(Lamps(), Following(5, 3, 20), Timing())
        controller.step(0.0)
        controller.warn(1.0, True)
        status = status_of(controller.snapshot(1.1))
        self.assertTrue(status["notzustand"])
        self.assertEqual(status["phase"], "Stoerung")
        self.assertEqual(set(status["signalbilder"].values()), {"GelbBlinken"})


class SocketTest(unittest.TestCase):
    def setUp(self):
        import tempfile
        from ..api import serve
        self.directory = tempfile.mkdtemp(prefix="ampel")
        self.path = os.path.join(self.directory, "ampel.sock")
        self.commands = []
        self.server = serve(self.path, lambda: {"phase": "Allrot", "an": True},
                            lambda switch, on: self.commands.append((switch, on)))

    def tearDown(self):
        self.server.shutdown()
        self.server.server_close()
        os.remove(self.path)
        os.rmdir(self.directory)

    def ask(self, method, route):
        connection = socket.socket(socket.AF_UNIX)
        connection.settimeout(3)
        connection.connect(self.path)
        connection.sendall(f"{method} {route} HTTP/1.0\r\n\r\n".encode())
        response = b""
        while True:
            chunk = connection.recv(4096)
            if not chunk:
                break
            response += chunk
        connection.close()
        head, _, body = response.partition(b"\r\n\r\n")
        return int(head.split()[1]), body

    def test_status_is_json(self):
        status, body = self.ask("GET", "/status")
        self.assertEqual(status, 200)
        self.assertEqual(json.loads(body), {"phase": "Allrot", "an": True})

    def test_switch_commands_reach_the_controller(self):
        for route, expected in (("/hauptschalter/aus", ("hauptschalter", False)),
                                ("/notschalter/an", ("notschalter", True))):
            with self.subTest(route=route):
                status, _ = self.ask("POST", route)
                self.assertEqual(status, 204)
                self.assertEqual(self.commands[-1], expected)

    def test_bad_requests_are_refused(self):
        self.assertEqual(self.ask("GET", "/nichts")[0], 404)
        self.assertEqual(self.ask("POST", "/lampe/an")[0], 404)
        self.assertEqual(self.ask("POST", "/notschalter/vielleicht")[0], 400)
        self.assertEqual(self.commands, [])

    def test_silent_client_does_not_block_the_server(self):
        from ..api import Handler
        original = Handler.timeout
        self.assertIsNotNone(original)
        Handler.timeout = 0.2
        self.addCleanup(setattr, Handler, "timeout", original)
        silent = socket.socket(socket.AF_UNIX)
        silent.connect(self.path)
        self.addCleanup(silent.close)
        self.assertEqual(self.ask("GET", "/status")[0], 200)

    def test_silent_clients_do_not_delay_others(self):
        import time
        silent = []
        for _ in range(3):
            client = socket.socket(socket.AF_UNIX)
            client.connect(self.path)
            silent.append(client)
            self.addCleanup(client.close)
        started = time.monotonic()
        self.assertEqual(self.ask("GET", "/status")[0], 200)
        self.assertLess(time.monotonic() - started, 1.0)

    def test_stale_socket_file_is_replaced(self):
        from ..api import serve
        self.server.shutdown()
        self.server.server_close()
        self.server = serve(self.path, lambda: {}, lambda switch, on: None)
        self.assertEqual(self.ask("GET", "/status")[0], 200)
