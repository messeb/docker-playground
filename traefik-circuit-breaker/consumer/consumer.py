"""
Consumer that repeatedly calls the API through Traefik and reports each result.

Circuit breaker states (from the consumer's perspective):
  CLOSED    — normal operation, requests reach the API
  OPEN      — Traefik returns 503 immediately, API never sees the request
              Traefik holds this state for fallbackDuration (5s), then…
  HALF-OPEN — Traefik forwards requests again to test if the backend recovered
              success → CLOSED   (traffic flows normally)
              failure → OPEN     (back to blocking, 5s timer resets)
"""

import os
import time
import datetime
import urllib.request
import urllib.error


URL           = os.getenv("API_URL",       "http://traefik-cb/api")
DELAY         = float(os.getenv("REQUEST_DELAY",  "0.3"))
REQUEST_COUNT = int(os.getenv("REQUEST_COUNT",    "60"))

# ── counters ──────────────────────────────────────────────────────────────────
ok        = 0
errors    = 0
cb_open   = 0

# ── circuit breaker state machine ─────────────────────────────────────────────
cb_state      = "CLOSED"
was_blocked   = False
open_at       = None   # wall-clock time when circuit last opened


def now():
    return datetime.datetime.now().strftime("%H:%M:%S")


def response_label(status):
    if status == "ok":  return "200 OK   "
    if status == "err": return "500 Error"
    if status == "cb":  return "503 ─────"
    return                     "??? ─────"


print(f"Sending {REQUEST_COUNT} requests to {URL}  (delay={DELAY}s)\n")
print(f"  Circuit breaker config: expression >50% 5xx  |  open for 5s  |  half-open for 5s\n")
print(f"  {'#':>3}  {'time':>8}  {'response':9}  circuit state")
print(f"  {'─'*3}  {'─'*8}  {'─'*9}  {'─'*50}")

for i in range(1, REQUEST_COUNT + 1):
    status = None
    try:
        with urllib.request.urlopen(URL, timeout=3) as resp:
            resp.read()
            status = "ok"
            ok += 1
    except urllib.error.HTTPError as e:
        if e.code == 503:
            status = "cb"
            cb_open += 1
        else:
            status = "err"
            errors += 1
    except Exception:
        status = "err"
        errors += 1

    # ── infer circuit state and annotate transitions ───────────────────────────
    note = ""
    if status == "cb":
        if not was_blocked:
            cb_state = "OPEN"
            open_at  = time.monotonic()
            note = "  ← OPEN: error ratio exceeded threshold, Traefik blocks all requests"
        was_blocked = True
    else:
        if was_blocked:
            elapsed = time.monotonic() - open_at if open_at else 0
            cb_state = "HALF-OPEN"
            note = f"  ← HALF-OPEN: fallbackDuration ({elapsed:.1f}s) elapsed, probe forwarded"
        elif cb_state == "HALF-OPEN":
            cb_state = "CLOSED"
            note = "  ← CLOSED: probe succeeded, circuit recovered"
        was_blocked = False

    print(f"  {i:>3}  {now()}  {response_label(status)}  {cb_state}{note}")
    time.sleep(DELAY)

# ── summary ───────────────────────────────────────────────────────────────────
total = ok + errors + cb_open
print(f"\n{'─'*52}")
print(f"  Total requests : {total}")
print(f"  200 OK         : {ok:>3}  ({ok/total*100:.0f}%)  reached API, success")
print(f"  500 Error      : {errors:>3}  ({errors/total*100:.0f}%)  reached API, failed")
print(f"  503 CB open    : {cb_open:>3}  ({cb_open/total*100:.0f}%)  blocked by Traefik, never reached API")
print(f"{'─'*52}")
