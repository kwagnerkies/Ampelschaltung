import json
import os
import socketserver
import threading
from http.server import BaseHTTPRequestHandler

from .signal import DIRECTION_NAMES, DIRECTIONS

SWITCHES = ["hauptschalter", "notschalter"]
POSITIONS = ["an", "aus"]


class Handler(BaseHTTPRequestHandler):
    def log_message(self, format, *args):
        pass

    def do_GET(self):
        if self.path != "/status":
            self.send_error(404)
            return
        body = json.dumps(self.server.state(), ensure_ascii=False).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_POST(self):
        parts = self.path.strip("/").split("/")
        if len(parts) != 2 or parts[0] not in SWITCHES:
            self.send_error(404)
            return
        if parts[1] not in POSITIONS:
            self.send_error(400, "stellung muss an oder aus sein")
            return
        self.server.command(parts[0], parts[1] == "an")
        self.send_response(204)
        self.end_headers()


def serve(path, state, command):
    if os.path.exists(path):
        os.remove(path)
    server = socketserver.UnixStreamServer(path, Handler)
    server.state = state
    server.command = command
    os.chmod(path, 0o660)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    return server


def status_of(snapshot):
    green_times = {}
    aspects = {}
    for direction in DIRECTIONS:
        name = DIRECTION_NAMES[direction]
        green_times[name] = round(snapshot.green[direction])
        aspects[name] = snapshot.aspects[direction].value
    return {
        "an": snapshot.on,
        "notzustand": snapshot.warning,
        "phase": snapshot.state.name(),
        "verlaengerungen": snapshot.following,
        "gruenzeiten_s": green_times,
        "signalbilder": aspects,
    }
