# Nginx Webserver

Nginx as a static file server in Docker, demonstrating the configuration patterns you'd use in a real setup: gzip compression, security headers, cache control for static assets, custom error pages, and a structured access log.

## Project structure

```
nginx-webserver/
├── compose.yml
├── Makefile
├── nginx.conf
└── html/
    ├── index.html          # Home page — links to all use cases
    ├── about.html          # Explains location block types and headers
    ├── 404.html            # Custom not-found page
    ├── 50x.html            # Custom server error page
    └── static/
        └── style.css       # Shared stylesheet (served with cache headers)
```

## Use cases

Each endpoint in this playground demonstrates a specific nginx feature. Open your browser's Network tab to inspect the response headers.

| Endpoint | Feature | Description |
|---|---|---|
| `http://localhost/` | Static page | Catch-all `location /` with `try_files` |
| `http://localhost/about.html` | Static page | Multi-page serving, location block types explained |
| `http://localhost/health` | Exact match | `location =` returns JSON without reading a file |
| `http://localhost/static/style.css` | Cache headers | Regex `location ~*` with `Cache-Control: public, immutable` |
| `http://localhost/missing` | Custom 404 | `error_page 404` with `internal` location |
| `http://localhost/error-500` | Custom 500 | `return 500` intercepted by `error_page 500` |

## nginx.conf features

### Location block types

nginx evaluates locations in a fixed priority order — exact before regex before prefix:

```nginx
# 1. Exact match (=) — fastest, for fixed paths like health checks
location = /health {
    return 200 '{"status":"ok"}';
}

# 2. Exact match (=) — simulates a 500 error, intercepted by error_page
location = /error-500 {
    return 500;
}

# 3. Regex match (~*) — case-insensitive, targets static asset extensions
location ~* \.(css|js|woff2?|ico|png|jpg|svg|webp)$ {
    expires 30d;
    add_header Cache-Control "public, immutable";
}

# 4. Prefix match (/) — catch-all fallback
location / {
    try_files $uri $uri/ =404;
}
```

### Gzip compression

```nginx
gzip            on;
gzip_vary       on;
gzip_comp_level 5;
gzip_types      text/plain text/css application/json
                application/javascript image/svg+xml font/woff2;
```

`text/html` is always compressed by nginx regardless of `gzip_types` — no need to add it explicitly.
`gzip_vary on` adds `Vary: Accept-Encoding` so CDNs cache compressed and uncompressed versions separately.

### Security headers

```nginx
add_header X-Content-Type-Options  "nosniff"       always;
add_header X-Frame-Options         "SAMEORIGIN"    always;
add_header X-XSS-Protection        "1; mode=block" always;
add_header Referrer-Policy         "same-origin"   always;
```

| Header | Purpose |
|---|---|
| `X-Content-Type-Options: nosniff` | Prevents browsers from MIME-sniffing away from the declared content type |
| `X-Frame-Options: SAMEORIGIN` | Blocks the page from being embedded in an iframe on a different origin (clickjacking protection) |
| `X-XSS-Protection: 1; mode=block` | Legacy XSS filter for older browsers |
| `Referrer-Policy: same-origin` | Only sends the `Referer` header on same-origin navigation |

### Cache control for static assets

```nginx
location ~* \.(css|js|woff2?|ico|png|jpg|svg|webp)$ {
    expires            30d;
    add_header         Cache-Control "public, immutable";
    access_log         off;
    try_files          $uri =404;
}
```

`immutable` tells the browser the file content will never change during the expiry window — it skips the conditional revalidation request entirely. Pair this with content-hashed filenames (e.g. `style.abc123.css`) in production.

### Custom error pages

```nginx
error_page 404             /404.html;
error_page 500 502 503 504 /50x.html;

location = /404.html { internal; }
location = /50x.html { internal; }
```

`internal` prevents clients from requesting the error pages directly — only nginx's own error handling can serve them. Requesting `/404.html` directly returns a 404 itself.

### Structured access log

```nginx
log_format playground '$remote_addr  $request  $status  '
                      '$body_bytes_sent bytes  $request_time s  '
                      '"$http_user_agent"';
```

Example output:
```
172.21.0.1  GET /static/style.css HTTP/1.1  200  1842 bytes  0.001 s  "curl/8.7.1"
172.21.0.1  GET /health HTTP/1.1            200  38 bytes    0.000 s  "curl/8.7.1"
172.21.0.1  GET /error-500 HTTP/1.1         500  177 bytes   0.000 s  "Mozilla/5.0"
```

Note: the `log_format` name must not be `main` — that name is already defined in nginx's built-in `nginx.conf` and cannot be redeclared in a `conf.d/` file.

## Usage

| Command | Description |
|---|---|
| `make up` | Start the container in the background |
| `make down` | Stop and remove the container |
| `make restart` | Restart the container |
| `make logs` | Follow nginx access and error logs |
| `make ps` | Show container status |
| `make open` | Open the webserver in the browser |
| `make headers` | Show response headers for key endpoints |
| `make help` | Show all available targets |

### Verify the configuration

After `make up`, run `make headers` to confirm gzip, cache, and security headers:

```
── / ──────────────────────────────────────────
HTTP/1.1 200 OK
Content-Encoding: gzip
X-Content-Type-Options: nosniff
X-Frame-Options: SAMEORIGIN
── /static/style.css ──────────────────────────
HTTP/1.1 200 OK
Cache-Control: public, immutable
Expires: Thu, 28 Apr 2026 00:00:00 GMT
── /health ────────────────────────────────────
{"status":"ok","service":"nginx-webserver"}
```

### Test error pages

```bash
# Custom 404 — missing path
curl -i http://localhost/missing

# Custom 500 — simulated server error
curl -i http://localhost/error-500

# Error pages are not directly accessible
curl -i http://localhost/404.html   # returns 404
curl -i http://localhost/50x.html   # returns 404
```

## Stop

```bash
make down
```
