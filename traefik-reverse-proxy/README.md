# Traefik Reverse Proxy

Traefik acting as a reverse proxy with path-based routing defined in a static YAML file. Routes, middlewares, and backend services are all declared explicitly in `traefik/dynamic/routes.yml` — no Docker socket access needed.

## Architecture

```
Client
  └─► Traefik (:80)
        ├─► /service-a  ──strip prefix──►  service-a (nginx, internal)
        └─► /service-b  ──strip prefix──►  service-b (nginx, internal)
```

Traefik watches the `dynamic/` directory for changes. Edit `routes.yml` and the new config is picked up live — no restart needed.

## Project structure

```
traefik-reverse-proxy/
├── compose.yml
├── Makefile
├── scripts/
│   └── status.py              # Traefik API status reporter
├── traefik/
│   ├── traefik.yml            # Static config: entrypoints + file provider
│   └── dynamic/
│       └── routes.yml         # Dynamic config: routers, middlewares, services
├── service-a/
│   └── html/index.html
└── service-b/
    └── html/index.html
```

## Key concepts

### Static config (`traefik.yml`)

Loaded once at startup. Defines the entrypoints (ports Traefik listens on) and which providers supply the dynamic routing config. Here the file provider watches a directory for YAML files.

### Dynamic config (`dynamic/routes.yml`)

Hot-reloaded whenever the file changes. Contains three sections:

**Routers** — match incoming requests and hand them off to a service:
```yaml
service-a:
  rule: "PathPrefix(`/service-a`)"   # match requests starting with /service-a
  entryPoints: [web]                 # listen on the :80 entrypoint
  middlewares: [strip-service-a]     # apply prefix stripping before forwarding
  service: service-a                 # forward to this service
```

**Middlewares** — transform requests in transit:
```yaml
strip-service-a:
  stripPrefix:
    prefixes: [/service-a]   # removes /service-a so the backend sees /
```

Without this, `GET /service-a/index.html` would be forwarded as-is and nginx would return 404 because it doesn't know about the `/service-a` prefix.

**Services** — define where to forward traffic:
```yaml
service-a:
  loadBalancer:
    servers:
      - url: "http://service-a:80"   # Docker DNS resolves service-a to the container(s)
```

Adding more `url` entries here enables load balancing across multiple instances.

## Usage

| Command | Description |
|---|---|
| `make up` | Start all containers in the background |
| `make down` | Stop and remove all containers |
| `make restart` | Restart all containers |
| `make logs` | Follow logs for all containers |
| `make logs-service s=service-a` | Follow logs for a specific service |
| `make ps` | Show running containers and their status |
| `make status` | Show Traefik routing status via the API |
| `make open-dashboard` | Open the Traefik dashboard in the browser |
| `make scale-a n=3` | Scale service-a to N replicas |
| `make scale-b n=3` | Scale service-b to N replicas |
| `make reload` | Reload Traefik config without downtime |
| `make help` | Show all available targets |

### `make status` output

```
  Dashboard   http://localhost:8080/dashboard/
  API         http://localhost:8080/api/http/routers

  4 routers   5 services   4 middlewares

Routers
  ✓  service-a@file     PathPrefix(`/service-a`)    [web]
  ✓  service-b@file     PathPrefix(`/service-b`)    [web]

Services
  service-a@file
    ·  http://service-a:80  [no health check]
  service-b@file
    ·  http://service-b:80  [no health check]

Middlewares
  strip-service-a@file    stripprefix
  strip-service-b@file    stripprefix
```

The `@file` suffix means the resource was loaded from the file provider.
The `·` indicator on services means no active health check is configured — this is normal for the file provider. Use `✓` / `✗` only when `healthCheck` is added to a service in `routes.yml`.

### Test each route

```bash
curl http://localhost/service-a
curl http://localhost/service-b
```

Open the Traefik dashboard at [http://localhost:8080](http://localhost:8080) to explore routers, services, and middlewares visually.

## Adding a new service

1. Add a container to `compose.yml` on the `proxy` network
2. Add a router, middleware, and service block to `traefik/dynamic/routes.yml`:

```yaml
routers:
  service-c:
    rule: "PathPrefix(`/service-c`)"
    entryPoints: [web]
    middlewares: [strip-service-c]
    service: service-c

middlewares:
  strip-service-c:
    stripPrefix:
      prefixes: [/service-c]

services:
  service-c:
    loadBalancer:
      servers:
        - url: "http://service-c:80"
```

Traefik picks it up instantly — no restart, no `make reload`.

## Scaling

Scale any backend service to multiple replicas — Docker's internal DNS round-robins traffic across them automatically:

```bash
make scale-a n=3
```

Traefik keeps using `http://service-a:80` in `routes.yml` unchanged. Docker resolves that hostname to whichever replica is available.

## Stop

```bash
make down
```
