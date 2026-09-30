import json
import os
import socket
import threading
from http.server import BaseHTTPRequestHandler, HTTPServer

from .signal import DIRECTION_NAMES, DIRECTIONS


class UnixServer(HTTPServer):
    address_family = socket.AF_UNIX



def serve(path, state, command):
    if os.path.exists(path):
        os.remove(path)

    class Handler(BaseHTTPRequestHandler):
        def log_message(self, *_):
            pass

        def do_GET(self):
            if self.path != "/status":
                self.send_error(404)
                return
            body = json.dumps(state(), ensure_ascii=False).encode()
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)

        def do_POST(self):
            parts = self.path.strip("/").split("/")
            if len(parts) != 2 or parts[0] not in ("hauptschalter", "notschalter"):
                self.send_error(404)
                return
            if parts[1] not in ("an", "aus"):
                self.send_error(400, "stellung muss an oder aus sein")
                return
            command(parts[0], parts[1] == "an")
            self.send_response(204)
            self.end_headers()

    server = UnixServer(path, Handler)
    os.chmod(path, 0o660)
    threading.Thread(target=server.serve_forever, daemon=True).start()
    return server


def status_of(snapshot):
    return {
        "an": snapshot.on,
        "notzustand": snapshot.warning,
        "phase": snapshot.state.name,
        "verlaengerungen": snapshot.following,
        "gruenzeiten_s": {
            DIRECTION_NAMES[d]: round(snapshot.green[d]) for d in DIRECTIONS
        },
        "signalbilder": {
            DIRECTION_NAMES[d]: snapshot.aspects[d].value for d in DIRECTIONS
        },
    }
