# Spring Boot + OpenTelemetry + Elastic APM

A Spring Boot application instrumented with OpenTelemetry that sends distributed traces to Elastic APM, visualized in Kibana. Demonstrates automatic HTTP tracing, custom child spans, slow transactions, and error traces — all without a Java agent.

## Architecture

```text
Spring Boot App
  └─► OTLP/HTTP ──► APM Server ──► Elasticsearch ──► Kibana (APM UI)
```

- **Spring Boot** auto-traces every HTTP request via Micrometer + OpenTelemetry bridge
- **APM Server** receives OTLP spans and indexes them in Elasticsearch
- **Kibana** visualizes traces, transaction durations, errors, and span waterfalls

## How tracing works

Spring Boot 3.x ships a native tracing integration via **Micrometer Tracing**. Adding two dependencies is enough — no Java agent, no manual SDK initialization:

```xml
<!-- Micrometer → OpenTelemetry bridge: auto-traces every HTTP request -->
<dependency>
    <groupId>io.micrometer</groupId>
    <artifactId>micrometer-tracing-bridge-otel</artifactId>
</dependency>
<!-- OTLP HTTP exporter: ships spans to APM Server -->
<dependency>
    <groupId>io.opentelemetry</groupId>
    <artifactId>opentelemetry-exporter-otlp</artifactId>
</dependency>
```

Custom child spans are created with the `ObservationRegistry` API:

```java
Observation.createNotStarted("fetch-order-1", registry).observe(() -> {
    // work here becomes a child span in the trace
});
```

## Project structure

```text
spring-opentelemetry-elastic-apm/
├── compose.yml                         # Elastic stack + Spring app
├── Makefile
├── .env                                # Credentials (ELASTIC_PASSWORD etc.)
├── Dockerfile                          # Multi-stage Maven build
└── src/main/
    ├── resources/application.properties
    └── java/net/messeb/OpenTelemetry/
        ├── OpenTelemetryApplication.java
        └── controller/
            ├── MyController.java       # Demo endpoints
            └── GlobalExceptionHandler.java
```

## Quick start

```bash
make up
```

First run builds the Spring image and pulls ~2 GB of Elastic images. The stack takes **2–3 minutes** to become fully ready.

Check readiness:

```bash
docker compose ps   # all containers should show "healthy"
```

Then generate sample traces:

```bash
make trace
```

## Kibana login

Open [http://localhost:5601](http://localhost:5601)

| Field | Value |
| --- | --- |
| Username | `elastic` |
| Password | value of `ELASTIC_PASSWORD` in `.env` (default: `elastic`) |

Navigate to **Observability → APM → spring-otel-demo** or use `make open-apm`.

## Demo endpoints

| Endpoint | Status | What it shows in APM |
| --- | --- | --- |
| `GET /api/hello` | 200 | Fast successful transaction |
| `GET /api/slow?ms=1200` | 200 | High-latency transaction (visible in latency chart) |
| `GET /api/orders` | 200 | Span waterfall with 3 child spans |
| `GET /api/orders/42` | 200 | Parameterized route — grouped as `GET /api/orders/{id}` |
| `GET /api/error` | 500 | Error trace + full stack trace in APM Errors tab |
| `GET /api/orders/999` | 404 | Not-found error trace |

## What to look at in Kibana

### Transactions list

**Observability → APM → spring-otel-demo → Transactions**

Shows all route groups with average latency, throughput, and error rate. `GET /api/slow` stands out with high latency.

### Trace waterfall

Click any transaction → **Trace sample** → expand the trace.

For `/api/orders`, the waterfall shows:

```text
GET /api/orders  (260ms total)
  └── fetch-orders
        ├── fetch-order-1  (80ms)
        ├── fetch-order-2  (120ms)
        └── fetch-order-3  (60ms)
```

### Errors

**Observability → APM → spring-otel-demo → Errors**

Lists every exception with its stack trace, grouped by exception class.

### Log correlation

Run `make logs-app` — every log line includes `traceId` and `spanId`:

```text
INFO [spring-otel-demo,4bf92f3577b34da6a3ce929d0e0e4736,00f067aa0ba902b7] orders called
```

Paste the `traceId` into Kibana → Discover to find all logs for a single request.

## Configuration

`.env` variables:

| Variable | Default | Description |
| --- | --- | --- |
| `STACK_VERSION` | `8.17.0` | Elastic stack version (all components must match) |
| `ELASTIC_PASSWORD` | `elastic` | Password for the `elastic` superuser |
| `KIBANA_SYSTEM_PASSWORD` | `kibana_system_password` | Internal password for Kibana→Elasticsearch connection |

`application.properties` tracing settings:

| Property | Default | Description |
| --- | --- | --- |
| `management.tracing.sampling.probability` | `1.0` | Fraction of requests to trace (1.0 = 100%) |
| `management.otlp.tracing.endpoint` | `http://localhost:8200/v1/traces` | OTLP endpoint — overridden by env var in Docker |

## Usage

| Command | Description |
| --- | --- |
| `make up` | Build the app and start the full stack |
| `make trace` | Send one request per endpoint to generate traces |
| `make open-apm` | Open the APM service view in the browser |
| `make open-kibana` | Open Kibana home |
| `make logs-app` | Follow Spring app logs (shows traceId per request) |
| `make logs` | Follow all service logs |
| `make down` | Stop containers |
| `make clean` | Stop and remove all containers, images, and volumes |

## Stop

```bash
make clean
```

`--volumes` in `make clean` also removes the Elasticsearch data volume — Kibana will require re-setup on next `make up`.
