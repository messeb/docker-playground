# Circuit Breaker with Traefik

Demonstrates Traefik's built-in circuit breaker middleware. A backend API is configured to fail a percentage of requests. A consumer calls it repeatedly through Traefik so you can observe the circuit breaker trip, hold, and recover in real time.

## Architecture

```text
Consumer ──► Traefik ──► circuit-breaker middleware ──► API (35% failure rate)
                                    │
                                    └─► 503 immediately when circuit is open
                                        (request never forwarded to API)
```

## Circuit breaker states

```text
          > 50% errors            fallbackDuration (5s)
 Closed ──────────────► Open ──────────────────────────► Half-Open
    ▲                                                         │
    └─────────────────────────────────────────────────────────┘
         probe succeeds → Closed       probe fails → Open
```

| State | What happens |
| --- | --- | 
| **Closed** | Requests forwarded to the API normally. Errors are counted. |
| **Open** | Traefik returns `503` immediately — API never sees the request. Holds for `fallbackDuration` (5s). |
| **Half-Open** | After 5s, Traefik forwards one probe request to test if the backend recovered. Success → Closed. Failure → Open again. |

The transition from Open → Half-Open is **time-based**, not request-based. Traefik waits exactly `fallbackDuration` seconds and then automatically allows a probe through.

## Project structure

```
traefik-circuit-breaker/
├── compose.yml
├── Makefile
├── traefik/
│   ├── traefik.yml             # Static config — file provider, dashboard
│   └── dynamic/
│       └── routes.yml          # Router, circuit-breaker middleware, service
├── api/
│   ├── Dockerfile
│   └── app.py                  # Python HTTP server with configurable failure rate
└── consumer/
    ├── Dockerfile
    └── consumer.py             # Makes N requests, reports state transitions live
```

## Quick start

```bash
make demo
```

Starts Traefik and the API, then runs the consumer. Output appears live as each request is made.

## What you will see

```bash
  Circuit breaker config: expression >50% 5xx  |  open for 5s  |  half-open for 5s

    #      time  response  circuit state
  ───  ────────  ─────────  ──────────────────────────────────────────────────────
    1  10:42:00  200 OK     CLOSED
    2  10:42:01  500 Error  CLOSED
    3  10:42:01  500 Error  CLOSED
    4  10:42:02  503 ─────  OPEN      ← OPEN: error ratio exceeded threshold, Traefik blocks all requests
    5  10:42:02  503 ─────  OPEN
    ...
   20  10:42:07  200 OK     HALF-OPEN ← HALF-OPEN: fallbackDuration (5.1s) elapsed, probe forwarded
   21  10:42:08  200 OK     CLOSED    ← CLOSED: probe succeeded, circuit recovered
   22  10:42:08  500 Error  CLOSED
   ...
```

**Key observations:**

- `503` lines: Traefik is blocking — the API container receives **no traffic**
- `HALF-OPEN` line: exactly `fallbackDuration` seconds after the circuit opened, Traefik forwards one probe
- `CLOSED` line: probe succeeded, normal traffic resumes
- If the probe fails, the circuit goes straight back to `OPEN` and the 5s timer resets

Verify that 503 responses never reach the API by checking `docker compose logs api` — you will see no log entries for those requests.

## Circuit breaker configuration

Defined in `traefik/dynamic/routes.yml`:

```yaml
middlewares:
  circuit-breaker:
    circuitBreaker:
      expression: "ResponseCodeRatio(500, 600, 0, 600) > 0.5"
      checkPeriod: 1s
      fallbackDuration: 5s
      recoveryDuration: 5s
```

| Parameter | Value | Description |
| --- | --- | --- |
| `expression` | `ResponseCodeRatio(500,600,0,600) > 0.5` | Trip when >50% of responses in the window are 5xx |
| `checkPeriod` | `1s` | How often the expression is re-evaluated |
| `fallbackDuration` | `5s` | How long the circuit stays Open before trying a probe |
| `recoveryDuration` | `5s` | How long Traefik waits in Half-Open before declaring recovery |

## Tuning

Edit environment variables in `compose.yml`:

| Variable | Default | Description |
| --- | --- | --- |
| `FAILURE_RATE` | `0.35` | Fraction of API requests that return 500 (0.0–1.0) |
| `REQUEST_COUNT` | `60` | Total requests the consumer makes |
| `REQUEST_DELAY` | `0.3` | Seconds between requests |

**Experiments:**

- `FAILURE_RATE=0.0` — circuit never trips, all requests succeed
- `FAILURE_RATE=1.0` — circuit trips after the first window and never recovers (every probe fails)
- `FAILURE_RATE=0.6` — circuit trips quickly, recovers slowly (probes succeed 40% of the time)

## Traefik dashboard

Open [http://localhost:8080](http://localhost:8080) while the demo is running to see the circuit breaker middleware listed under HTTP → Middlewares.

## Stop

```bash
make clean
```
