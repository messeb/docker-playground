# Docker Build-Time Secret Handover

Demonstrates **when and why** to use `RUN --mount=type=secret` instead of `--build-arg`
for secrets that are only needed during the image build — not at runtime.

## The Scenario

Your app requires a private config file that lives behind an authenticated HTTP endpoint
(a private registry, a license server, an internal artifact store).
The token is needed **only to fetch the file during `docker build`**.
The running container only needs the downloaded file, not the token itself.

```
build time:  token → curl → private registry → private-config.txt (baked into image)
run time:    container reads private-config.txt — token is gone
```

## Why Not `--build-arg`?

Build arguments are written into the image layer history permanently:

```bash
docker history --no-trunc app-insecure
# ... RUN curl -H "Authorization: Bearer super-secret-registry-token-xyz789" ...
```

Anyone with `docker pull` access to the image can recover the token.
It will also appear in CI logs if the build output is not masked.

## The Secure Approach

`RUN --mount=type=secret` mounts a file into a single `RUN` step only.
It is **never written to any layer** and does not appear in `docker history` or `docker inspect`.

```dockerfile
RUN --mount=type=secret,id=registry_token \
    curl -sSf \
      -H "Authorization: Bearer $(cat /run/secrets/registry_token)" \
      https://private-registry/private-config.txt \
      -o /etc/app/private-config.txt
```

```bash
DOCKER_BUILDKIT=1 docker build \
  --secret id=registry_token,src=secrets/registry_token.txt \
  -t my-app .
```

> **Note (Docker Desktop / macOS):** During `docker build`, BuildKit runs in an isolated
> context that does not support attaching to arbitrary Docker networks.
> The private registry is published on the host and reached via `host.docker.internal:19080`.

## Project Structure

```
.
├── Dockerfile              # Secure build — uses --mount=type=secret
├── Dockerfile.insecure     # Insecure build — uses --build-arg (for comparison)
├── entrypoint.sh           # Prints the fetched config at runtime
├── private-registry/
│   ├── Dockerfile          # Simulated private asset server (Python)
│   └── server.py           # Requires Bearer token to serve private-config.txt
├── secrets/
│   └── registry_token.txt  # The token (never baked into any image)
└── Makefile
```

## Running the Demo

```bash
# Full demo: build both images, inspect layers, run both containers
make demo

# Or step by step:
make build-secure    # build using --mount=type=secret
make build-insecure  # build using --build-arg
make verify          # check whether token appears in docker history
make run-secure      # run the secure container
make run-insecure    # run the insecure container

make clean           # remove all containers, networks, and images
```

## Expected Output of `make verify`

```
  [app-secure]    PASS — token NOT found in layer history
  [app-insecure]  FAIL — token found in layer history (as expected)
```

## Other Real-World Use Cases

| Situation | What the secret is |
|---|---|
| `pip install` from private PyPI | `~/.netrc` or index URL with credentials |
| `npm install` from private registry | `.npmrc` with `//registry:_authToken=...` |
| `go get` from private module proxy | `GONOSUMCHECK` + `GOAUTH` token |
| Download licensed SDK / binary | API key or signed URL |
| Clone private Git repo | SSH key or GitHub token |
| Fetch config from Vault / AWS SSM | Vault token or AWS credentials |

In every case the pattern is the same: the secret grants access to fetch an artifact.
Once the artifact is in the layer, the secret has no further role.

## How Build Secrets Work Under the Hood

BuildKit handles secrets entirely outside the layer system. When a `RUN --mount=type=secret` step executes:

1. The secret file is read from the **host** — never from the build context
2. It is mounted as a **tmpfs** at `/run/secrets/<id>` — in-memory only, never written to disk inside the container
3. The `RUN` step executes
4. The mount is **unmounted and discarded** before the layer is committed

The resulting layer contains only the side-effects (e.g. the downloaded file), not the secret.

## Mount Options

```dockerfile
# Default path is /run/secrets/<id>, but can be overridden
RUN --mount=type=secret,id=token,target=/etc/myapp/token \
    cat /etc/myapp/token

# Fail the build if the secret is not provided
RUN --mount=type=secret,id=token,required=true \
    cat /run/secrets/token
```

## Passing the Secret

```bash
# From a file
docker build --secret id=registry_token,src=secrets/registry_token.txt .

# From an environment variable
docker build --secret id=registry_token,env=REGISTRY_TOKEN .
```

## Multiple Secrets in One Step

```dockerfile
RUN --mount=type=secret,id=npm_token \
    --mount=type=secret,id=github_token \
    npm install && git clone ...
```

## What It Does Not Protect Against

- **A malicious `RUN` step** that explicitly writes the secret into the layer: `cat /run/secrets/token > /baked-token` — Docker cannot prevent this, the Dockerfile author is responsible
- **Runtime secrets** — this feature is build-time only; for runtime use Docker Swarm secrets or Kubernetes secrets mounted as volumes

## Comparison

| | `ARG` / `ENV` | `--mount=type=secret` |
|---|---|---|
| Visible in `docker history` | **yes** | no |
| Visible in `docker inspect` | **yes** (ENV) | no |
| Available at runtime | only if `ENV` used | no |
| Requires BuildKit | no | **yes** |
| Source can be a file or env var | no (string only) | **both** |
