#!/usr/bin/env python3
"""
Simulates a private asset registry that requires Bearer token authentication.
Serves a "premium config" file only to authenticated clients.
"""
from http.server import HTTPServer, BaseHTTPRequestHandler
import os

VALID_TOKEN = os.environ.get("REGISTRY_TOKEN", "changeme")

PRIVATE_CONFIG = b"""\
# Private Config (fetched during build)
FEATURE_FLAGS=premium,advanced-analytics
LICENSE_KEY=XXXX-YYYY-ZZZZ-9999
MAX_CONNECTIONS=500
"""


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        auth = self.headers.get("Authorization", "")
        if auth != f"Bearer {VALID_TOKEN}":
            self.send_response(401)
            self.send_header("Content-Type", "text/plain")
            self.end_headers()
            self.wfile.write(b"Unauthorized")
            return

        if self.path == "/private-config.txt":
            self.send_response(200)
            self.send_header("Content-Type", "text/plain")
            self.end_headers()
            self.wfile.write(PRIVATE_CONFIG)
            return

        self.send_response(404)
        self.end_headers()

    def log_message(self, format, *args):
        pass  # suppress request logs


if __name__ == "__main__":
    server = HTTPServer(("0.0.0.0", 8080), Handler)
    print("Private registry listening on :8080", flush=True)
    server.serve_forever()
