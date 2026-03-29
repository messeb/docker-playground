"""
Backend API that simulates failures.

FAILURE_RATE env var (0.0–1.0) controls how often /api returns 500.
/api/health always returns 200 — used to verify the API is reachable.
"""

import os
import random
import datetime
from http.server import BaseHTTPRequestHandler, HTTPServer


FAILURE_RATE = float(os.getenv("FAILURE_RATE", "0.0"))
PORT = int(os.getenv("PORT", "8000"))


def now():
    return datetime.datetime.now().strftime("%H:%M:%S")


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == "/api/health":
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            self.wfile.write(b'{"status":"ok"}\n')
            return

        # Fault-injecting endpoint — simulates a flaky backend
        if random.random() < FAILURE_RATE:
            self.send_response(500)
            self.send_header("Content-Type", "text/plain")
            self.end_headers()
            self.wfile.write(b"Internal Server Error (simulated)\n")
            print(f"[{now()}] [api] 500  {self.path}")
        else:
            self.send_response(200)
            self.send_header("Content-Type", "text/plain")
            self.end_headers()
            self.wfile.write(b"OK\n")
            print(f"[{now()}] [api] 200  {self.path}")

    def log_message(self, fmt, *args):
        pass  # handled in do_GET


if __name__ == "__main__":
    print(f"[{now()}] [api] Starting on :{PORT}  failure_rate={FAILURE_RATE:.0%}")
    HTTPServer(("", PORT), Handler).serve_forever()
