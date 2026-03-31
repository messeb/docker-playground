# 🏦 API Keycloak Security

A Go banking REST API secured with Keycloak, backed by PostgreSQL. Demonstrates two complementary security layers: **identity via Keycloak** (login, roles) and **token confidentiality via JWE** (access tokens are encrypted so only the API can read them).

## Architecture

```
  User
   │
   ├─ 1. POST /realms/banking/.../token  (username + password)
   │         │
   │         │  Keycloak fetches API public key from
   │         │  GET http://api:8080/.well-known/jwks.json
   │         │  then encrypts the token:
   │         │
   │         └──▶  JWE: header.encryptedKey.iv.ciphertext.tag
   │                    (RSA-OAEP key wrap + AES-256-GCM content)
   │
   └─ 2. GET /api/v1/accounts/me
          Authorization: Bearer <JWE token>
                │
                ▼
         ┌──────────────┐   decrypt JWE      ┌────────────────┐
         │  Keycloak    │   (private key)    │   Go API       │
         │              │──── JWKS keys ────▶│                │
         │  Issues      │   (on startup,     │  1. decrypt    │
         │  encrypted   │    cached)         │     JWE token  │
         │  JWE tokens  │                    │  2. validate   │
         │              │                    │     inner JWS  │
         └──────────────┘                    │  3. read claim │
                                             │  bank_account_ │
                                             │  number ───┐   │
                                             └────────────┼───┘
                                                          │
                                                          ▼
                                               ┌─────────────────┐
                                               │   PostgreSQL    │
                                               │                 │
                                               │  bank_accounts  │
                                               │  account_number │
                                               │  balance        │
                                               │                 │
                                               │  transactions   │
                                               └─────────────────┘
```

## Token encryption (JWE)

Standard JWTs are only **signed** — the payload is base64-encoded and readable by anyone who holds the token:

```
JWS (signed only):   header.payload.signature
                              ↑
                     base64url-decoded by anyone
```

This project adds **encryption** on top: Keycloak wraps the signed token in a JWE so the payload is opaque to the client:

```
JWE (signed + encrypted):   header.encryptedKey.iv.ciphertext.tag
                                                  ↑
                                     readable only by the API (holds the private key)
```

### How the key exchange works

| Step | Who | What |
|------|-----|------|
| Startup | API | Loads RSA private key from `API_PRIVATE_KEY_BASE64` in `.env` |
| Startup | API | Exposes public key at `GET /.well-known/jwks.json` |
| Login | Keycloak | Fetches `http://api:8080/.well-known/jwks.json` (one-time per key) |
| Login | Keycloak | Encrypts access token with RSA-OAEP + AES-256-GCM → JWE |
| API call | API | Decrypts JWE with private key → inner JWS |
| API call | API | Validates inner JWS signature with cached Keycloak JWKS |
| API call | API | Reads `bank_account_number` claim → looks up account in PostgreSQL |

The API is the **only party that can read token claims**. The client that holds the token cannot decode it.

### Key design decisions

| Concern | Solution |
|---------|----------|
| Token confidentiality | JWE (RSA-OAEP + AES-256-GCM): payload encrypted for API eyes only |
| JWT validation | Keycloak JWKS cached at startup, refreshed every 5 min in background |
| Keycloak coupling | Zero per-request Keycloak calls — JWKS fetched once at startup |
| User → account link | `bank_account_number` Keycloak user attribute → JWT claim → DB lookup |
| Balance safety | `SELECT ... FOR UPDATE` row lock inside a DB transaction |
| SQL injection | All queries use `$1`/`$2` pgx parameterized statements |
| Admin access | Two-layer middleware: JWT validity + realm role check |

## Project structure

```
api-keycloak-security/
├── compose.yml                      # Services: Keycloak, PostgreSQL, Go API
├── Makefile                         # Lifecycle + demo workflow targets
├── .env                             # Credentials + RSA private key (demo key)
├── api/
│   ├── Dockerfile                   # Multi-stage Go build, non-root user
│   ├── go.mod / go.sum
│   ├── cmd/
│   │   ├── server/main.go           # Wiring: config, pool, JWKS cache, routes
│   │   └── decode-token/main.go     # CLI tool: decrypt + inspect a JWE token
│   └── internal/
│       ├── auth/
│       │   ├── jwks.go              # Keycloak JWKS cache (lestrrat-go/jwx/v2)
│       │   ├── encryption.go        # RSA key loading + JWE decryption
│       │   └── middleware.go        # JWT + JWE middleware, role enforcement
│       ├── config/config.go         # Env-var loading + PEM decoding
│       ├── model/                   # BankAccount, Transaction structs
│       ├── repository/              # SQL queries (pgx/v5, parameterized)
│       ├── service/account.go       # Business logic, DB transactions
│       └── handler/
│           ├── jwks.go              # GET /.well-known/jwks.json
│           └── ...                  # Account, deposit, withdraw handlers
├── database/init.sql                # Schema + seed data (Alice + Bob)
└── keycloak/realm-export.json       # Banking realm, client, demo users + mapper
```

## Quick start

```bash
make up
```

Keycloak takes ~40–60 seconds on first run to import the realm. Watch with `make logs`.

### Login credentials

**Keycloak admin console** — `http://localhost:8180`

| Field | Value |
|-------|-------|
| Username | `admin` |
| Password | `admin` |

**Demo bank users** (login via token endpoint):

| User | Password | Roles | Bank account |
|------|----------|-------|--------------|
| `alice` | `alice123` | user | BANK-0001-2024 (balance 2,500) |
| `bob` | `bob123` | user, admin | BANK-0002-2024 (balance 5,000) |

### Get an access token

```bash
TOKEN=$(curl -sf \
  -X POST http://localhost:8180/realms/banking/protocol/openid-connect/token \
  -d "grant_type=password&client_id=banking-api&username=alice&password=alice123" \
  | python3 -c "import sys,json; print(json.load(sys.stdin)['access_token'])")
```

Or use the Makefile shorthand:
```bash
make token-alice    # prints full JSON response (access_token, expires_in, …)
make token-bob      # bob also has the 'admin' role
```

The returned token is a **JWE** — five dot-separated parts, payload not human-readable.

### Decrypt and inspect a token

```bash
make decrypt-token TOKEN=$TOKEN
```

Output:
```
─────────────────────────────────────────────
TOKEN TYPE  : JWE (encrypted)
ALGORITHM   : RSA-OAEP (key wrap) + A256GCM (content encryption)
KEY ID      : <sha256-thumbprint>
─────────────────────────────────────────────
DECRYPTED CLAIMS:
─────────────────────────────────────────────
{
  "bank_account_number": "BANK-0001-2024",
  "email": "alice@example.com",
  "preferred_username": "alice",
  "realm_access": { "roles": ["user"] },
  "sub": "...",
  "exp": 1234567890,
  ...
}
```

### Call the API

```bash
make me           TOKEN=$TOKEN    # GET  /api/v1/accounts/me
make deposit      TOKEN=$TOKEN    # POST /api/v1/accounts/me/deposit  { amount: 100 }
make withdraw     TOKEN=$TOKEN    # POST /api/v1/accounts/me/withdraw { amount: 50 }
make transactions TOKEN=$TOKEN    # GET  /api/v1/accounts/me/transactions
```

Admin endpoint (requires bob's token):
```bash
BOB=$(curl -sf -X POST http://localhost:8180/realms/banking/protocol/openid-connect/token \
  -d "grant_type=password&client_id=banking-api&username=bob&password=bob123" \
  | python3 -c "import sys,json; print(json.load(sys.stdin)['access_token'])")

make create-account TOKEN=$BOB ACCOUNT=BANK-0003-2024 NAME="Charlie"
```

## API reference

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/health` | — | Liveness probe |
| `GET` | `/.well-known/jwks.json` | — | API public key (consumed by Keycloak) |
| `GET` | `/api/v1/accounts/me` | user | Own account + balance |
| `POST` | `/api/v1/accounts/me/deposit` | user | `{ "amount": 100.0 }` |
| `POST` | `/api/v1/accounts/me/withdraw` | user | `{ "amount": 50.0 }` → 422 if insufficient |
| `GET` | `/api/v1/accounts/me/transactions` | user | `?limit=20&offset=0` |
| `POST` | `/api/v1/admin/accounts` | admin | `{ "account_number": "BANK-0003-2024", "owner_name": "Name" }` |

## Services

| Service | URL | Credentials |
|---------|-----|-------------|
| Go API | http://localhost:8081 | Bearer JWE token |
| Keycloak admin | http://localhost:8180 | admin / admin |
| PostgreSQL | localhost:5432 | bankuser / bankpassword |

## Key management

The demo ships with a pre-generated RSA key in `.env`. To rotate it:

```bash
make generate-keys    # generates new RSA 2048 key, updates .env
make clean            # full reset (Keycloak re-imports realm with new jwks.url)
make up               # fresh start with new key
```

After rotation, Keycloak fetches the new public key on the first login and issues tokens encrypted with the new key.
