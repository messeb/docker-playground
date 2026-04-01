# 🐳 Docker Playground

Hands-on Docker examples for real-world patterns — networking, proxying, resilience, security, observability, and more. Each project is self-contained and runnable with a single command.

## 📦 Projects

### 🌐 Web & Proxy

| Project | What it demonstrates |
| --- | --- |
| [🔀 Traefik Reverse Proxy](./traefik-reverse-proxy/) | Path-based routing (`/service-a`, `/service-b`) with Traefik using a static YAML file provider. No Docker socket required. |
| [⚡ Traefik Circuit Breaker](./traefik-circuit-breaker/) | Circuit breaker middleware that automatically stops forwarding requests to a failing backend and recovers after a timeout. |
| [⚖️ HAProxy Load Balancer](./haproxy-http-loadbalancer/) | Round-robin load balancing across multiple HTTP backends with HAProxy. |
| [📄 Nginx Webserver](./nginx-webserver/) | Static file server with gzip compression, security headers, cache control, custom error pages, and structured access logs. |
| [🧩 Varnish ESI](./varnish-edge-side-include/) | Edge Side Includes — assembling HTML pages from cached components using Varnish. |
| [🚦 Queue-Based Site Access](./queue-based-site-access/) | Virtual waiting room in front of a protected page — Redis-backed FIFO queue, live position polling, ETA, atomic slot claiming, and a bypass-proof architecture where the target has no published port. |

### 🗄️ Data

| Project | What it demonstrates |
| --- | --- |
| [🍃 MongoDB Prefilled](./mongodb-prefilled/) | MongoDB container that seeds a collection from a JSON file on startup. |
| [🐘 PostgreSQL Read Replicas](./postgresql-db-replicas/) | Primary + two read replicas with streaming replication — ready to test read scaling. |

### 🔒 Security

| Project | What it demonstrates |
| --- | --- |
| [🛡️ Secure Docker Container](./secure-docker-container/) | Multi-stage build, non-root user, read-only filesystem, dropped capabilities — security hardening checklist in one Dockerfile. |
| [🔑 Build-Time Secret Handover](./secret-handover/) | `RUN --mount=type=secret` vs `--build-arg` — why secrets passed as build args end up in the image history and how to avoid it. |
| [🏦 API Keycloak Security](./api-keycloak-security/) | Go banking REST API secured with Keycloak (OIDC login, roles) and JWE token encryption — access tokens are RSA-encrypted so only the API can read the claims. |

### 📡 Messaging & Observability

| Project | What it demonstrates |
| --- | --- |
| [🐇 RabbitMQ Producer / Consumer](./rabbitmq-producer-consumer/) | Work queue pattern with two Python services — one publishing messages, one consuming them. |
| [📊 Spring Boot + OpenTelemetry + Elastic APM](./spring-opentelemetry-elastic-apm/) | Distributed tracing from a Spring Boot app through OpenTelemetry to Elastic APM and Kibana. |

### ☸️ Kubernetes

| Project | What it demonstrates |
| --- | --- |
| [⎈ Local Kubernetes + ArgoCD](./local-kubernetes-setup/) | Fully local GitOps environment with kind, ArgoCD, and a local Git server — no cloud account needed. |

---

## 🚀 Quick start

Every project has a `Makefile`. The common entry point is always:

```bash
make        # show available targets
make demo   # or: make up
```

---

## 🔍 Checkout a single project

You don't need to clone the entire repository. Use Git sparse checkout to get only the project you want:

```bash
git clone --no-checkout --depth 1 git@github.com:messeb/docker-playground.git
cd docker-playground
git sparse-checkout init --cone
git sparse-checkout set traefik-circuit-breaker
git checkout main
```

Replace `traefik-circuit-breaker` with any project folder name. To add more projects later:

```bash
git sparse-checkout add nginx-webserver
```

---

## 💡 Tips & Tricks

**Port conflicts** — each project uses its own ports. Check the `compose.yml` before running two projects at the same time. Common ports used: `80`, `8080`, `5672`, `15672`, `5601`.

**Named networks** — projects that use Traefik define a named Docker network (`proxy`, `cb-net`, …). Run `docker network ls` if you see unexpected network warnings after `make down`.

**Rebuilding images** — if you change application code, pass `--build` to rebuild:
```bash
docker compose up --build
```
Or use `make clean` (where available) to remove cached images entirely.

**Following logs** — attach to a specific service without restarting it:
```bash
docker compose logs -f <service-name>
```

**Inspecting a container** — open a shell inside a running container:
```bash
docker compose exec <service-name> sh
```

**Stopping cleanly** — `make down` (or `docker compose down`) stops and removes containers but keeps images. Add `--rmi local` to also remove locally built images, or `--volumes` to wipe named volumes (e.g. database data).
