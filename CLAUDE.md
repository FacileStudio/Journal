# Journal

Centralized logging service for the Facile Suite. Facile apps POST structured log entries to
a Go API; a SvelteKit dashboard searches, filters, and live-tails them. Self-hosted,
Docker-deployed.

## Tech Stack

| Layer | Tech |
|-------|------|
| API | Go 1.24, Chi router, GORM, PostgreSQL 16 (full-text search via `tsvector` + GIN) |
| Client | SvelteKit 5 (Svelte 5 runes), Tailwind CSS 4, Bun, adapter-node |
| Auth | Dashboard: email/password accounts, DB-backed sessions (Argon2id, 30-day token). Ingest: per-app API keys (SHA256-hashed, admin-managed) with optional legacy static `INGEST_TOKEN`. Mirrors the Nuage pattern. |
| Infra | Docker Compose, Traefik (production), Dokploy |

## Key Commands

### Docker (full stack)

```sh
cp .env.example .env
docker compose up --build                                          # production: no host ports published
docker compose -f docker-compose.yml -f docker-compose.dev.yml up  # dev: 127.0.0.1:3000/4010/5432
```

### Local Development

```sh
# 1. Start Postgres
docker compose -f docker-compose.yml -f docker-compose.dev.yml up journal-db -d

# 2. API (port 4010)
cd apps/api
cp .env.example .env
go run .

# 3. Client (port 5173)
cd apps/client
bun install
bun run dev
```

### Ingest a test log

```sh
curl -X POST http://localhost:4010/ingest \
  -H "Authorization: Bearer change-me" \
  -H "Content-Type: application/json" \
  -d '{ "app": "nuage", "level": "error", "message": "upload failed", "meta": { "file_id": 42 } }'
```

### Client checks

```sh
cd apps/client
bun run check                      # svelte-check + TypeScript
bun run build                      # production build
```

## Project Structure

```
Journal/
  docker-compose.yml               # db, api, client — production shape, no host ports
  docker-compose.dev.yml           # opt-in (-f) local dev: publishes ports on 127.0.0.1
  .env.example                     # production env template
  apps/
    api/
      main.go                      # entrypoint, router + middleware stack, retention job, route registration
      internal/
        env/                       # config loading from env vars
        database/                  # GORM Postgres connection (pool: 10 open / 5 idle / 30m lifetime)
        httpjson/                  # JSON decode/encode + error helpers
        errors/                    # typed errors -> HTTP status mapping
        logger/                    # structured slog logging
        authcrypto/                # Argon2id password hashing + session/API token gen/hash, strict Bearer parsing
        authcontext/               # request-scoped authenticated identity + ingest key scope
        middleware/                # CORS, security headers, request logging, realip, ingest auth, RequireAuth, RequireAdmin
      schemas/                     # GORM models (log_entry, user, session, api_key, saved_query, alert_rule) + Migrate
      modules/
        auth/                      # /auth/{config,register,login,logout,me} — sessions
        ingest/                    # POST /ingest (single + batch, gzip), per-app key or legacy token
        logs/                      # GET /logs, /logs/histogram, /logs/{id}/context, GET /apps — session-protected
        apikeys/                   # /apikeys CRUD — session + admin only
        queries/                   # /queries CRUD (saved filter sets) — session-protected
        alerts/                    # /alerts CRUD + 60s webhook evaluator — session + admin only
    collector/                     # optional sidecar: tails all Docker containers via docker.sock, ships to /ingest
    client/
      src/
        hooks.server.ts            # security headers on all responses (CSP lives in svelte.config.js)
        lib/backend.ts             # typed API client (auth, logs, histogram, context, api keys)
        lib/auth.ts                # localStorage session token (journal.token)
        routes/
          login/+page.svelte       # sign in / register (redirects authed users to /)
          (app)/+layout.svelte     # auth guard — redirects to /login, exposes user via context
          (app)/+page.svelte       # dashboard: filters, saved queries, time range, histogram, live tail (pause/gap markers), pivots, context panel
          (app)/keys/+page.svelte  # API key management (admin only)
          (app)/alerts/+page.svelte # alert rules management (admin only)
          api/[...path]/           # reverse proxy to Go API (dev plumbing — prod bypasses it, see Architecture)
      static/                      # favicon, logo, fonts, vendored iconify-icon script
```

## Architecture

```
Facile apps ──POST /ingest──▶ Go API (:4010) ──▶ Postgres
Browser ──▶ SvelteKit (:3000) ──/api/*──▶ Go API (:4010)
```

In production, Traefik routes `journal.facile.studio/api/*` (stripprefix) **directly to the
Go API** and everything else to the SvelteKit client — the client's `/api/[...path]` reverse
proxy only carries traffic in local dev. The Go API's own middleware (CORS, security headers,
rate limits) is the real browser-facing perimeter; hardening in the SvelteKit proxy does not
apply to prod traffic. Postgres is internal with hardcoded credentials and no published ports.

## Environment Variables

| Variable | Description | Default |
|---|---|---|
| `DATABASE_URL` | Postgres connection string | `postgres://journal:journal@localhost:5432/journal?sslmode=disable` |
| `PORT` | API listen port | `4010` |
| `LOG_LEVEL` | `debug`, `info`, `warn`, `error` | `info` |
| `INGEST_TOKEN` | Legacy shared ingest token (unscoped). Empty disables it — per-app API keys are the primary ingest auth | — (empty) |
| `RETENTION_DAYS` | Delete log entries older than N days (hourly job); `0` keeps forever | `90` |
| `ALLOW_REGISTRATION` | `false` locks dashboard sign-ups (first account always allowed) | `true` |
| `ALLOWED_ORIGINS` / `DOMAINS` | Comma-separated CORS origins | — |
| `ORIGIN` | Public client URL — consumed by SvelteKit adapter-node (CSRF), not the Go API | `http://localhost:3000` |

## Schema

Table `log_entries`:

| Column | Type | Notes |
|---|---|---|
| `id` | bigint PK | |
| `app` | text | source app name, indexed |
| `level` | text | `debug`/`info`/`warn`/`error`, default `info`, indexed |
| `message` | text | |
| `meta` | jsonb | nullable, arbitrary structured context |
| `created_at` | timestamptz | log's own time (client `ts` or server now), indexed |
| `received_at` | timestamptz | server receipt time, autoCreateTime |
| `search` | tsvector | generated from `message`, GIN-indexed |

Extra indexes: GIN on `search`, composite `(app, created_at DESC)`, composite
`(created_at DESC, id DESC)` (keyset cursor + context queries), partial expression btree on
`(meta->>'request_id') WHERE meta ? 'request_id'`. `schemas.Migrate` runs `AutoMigrate` then
raw SQL for the generated column + indexes (GORM can't express a generated `tsvector` column
or a `DESC` composite index).

Table `api_keys`: `id` PK, `app` text, `prefix` text (display only), `key_hash` text unique
(SHA256 hex of the full token), `created_at`, `revoked_at` nullable.

## API Contract

### Auth (`/auth/*`)

Dashboard accounts. DB-backed sessions: a random 32-byte token is returned to the client and
stored SHA256-hashed in `sessions`; passwords are Argon2id. The first account created becomes
admin (guarded by `pg_advisory_xact_lock` inside the register transaction). The token is sent
as `Authorization: Bearer <token>` (scheme required, case-insensitive) and the client keeps it
in `localStorage` (`journal.token`). Login opportunistically deletes expired session rows.

- `GET /auth/config` → `{ "allow_registration": bool }` (drives the register tab)
- `POST /auth/register` → `{ token, user }` (201). Body `{ email, name?, password }`, password ≥ 12 chars. Locked once accounts exist if `ALLOW_REGISTRATION=false` (first account always allowed). Duplicate email → 409.
- `POST /auth/login` → `{ token, user }`. Body `{ email, password }`.
- `POST /auth/logout` (Bearer) → deletes the session.
- `GET /auth/me` (Bearer) → `{ user }` (includes `is_admin`).

`GET /logs*` and `GET /apps` require a valid Bearer session token; `/apikeys*` additionally
requires `is_admin`. `/health` and `/ready` stay public and rate-limit exempt.

Rate limits: login/register 20/min per IP per endpoint; ingest 600/min per token hash;
session routes 300/min per IP. Client IP honors the last `X-Forwarded-For` hop only when the
peer is loopback/private (Traefik), so it can't be spoofed from outside.

### `POST /ingest`

Auth: Bearer token, either a **per-app API key** (`journal_<app>_…`) or the legacy shared
`INGEST_TOKEN` (if configured). Per-app keys are scoped: each entry's `app` must be empty
(filled with the key's app) or equal to it, else 400. Legacy token is unscoped (`app`
required per entry). No valid credential → 401.

Single entry **or** batch (max 1000 entries, else 400). `Content-Encoding: gzip` accepted
(8MB raw cap, 32MB decompressed cap → 413). Entry fields: `app` (see above), `level`
(optional, default `info`), `message` (required, truncated at 64KB on a rune boundary with
`" [truncated]"` appended), `ts` (optional RFC3339 → `created_at`; more than 5 min in the
future → server receipt time), `meta` (optional object). Rate-limited responses are 429 with
`Retry-After: 60` — shippers should buffer and retry on 429/5xx and drop on other 4xx.

```jsonc
{ "app": "nuage", "level": "error", "message": "boom", "meta": { "k": "v" } }
// or
{ "entries": [ { "app": "opus", "message": "task created" } ] }
```

Response `201`: `{ "ingested": <n> }`. An explicit `{ "entries": [] }` → `{ "ingested": 0 }`.

### `GET /logs`

Query params: `app`, `level` (repeatable or CSV), `q` (full-text via
`websearch_to_tsquery('simple', q)`), `request_id` (matches `meta->>'request_id'`),
`since`/`until` (RFC3339 on `created_at`), `limit` (default 100, max 1000), and keyset cursor
`before_ts` (RFC3339Nano) + `before_id` (int64) — both or neither, predicate
`(created_at, id) < (?, ?)`. Ordered `created_at desc, id desc`.

Response: `{ "entries": [...], "next_before": { "ts", "id" } | null }`.

### `GET /logs/histogram`

Same filters as `/logs` (minus cursor/limit). Defaults: `until` = now, `since` = until − 24h.
Server picks the smallest bucket from {1m, 5m, 15m, 1h, 6h, 1d} giving ≤ 90 buckets.

Response: `{ "bucket_seconds": n, "buckets": [ { "ts", "counts": { "error": n, ... } } ] }` —
empty buckets and zero levels omitted (client fills gaps).

### `GET /logs/{id}/context?before=50&after=50`

Unfiltered stream around one entry (defaults 50, max 200 each; 404 unknown id). Response:
`{ "entries": [...], "anchor_id" }` sorted `created_at desc, id desc`, anchor included.

### `/apikeys` (session + admin)

- `GET /apikeys` → `{ "keys": [ { "id", "app", "prefix", "created_at", "revoked_at" } ] }`
- `POST /apikeys` body `{ "app" }` (`^[a-z0-9][a-z0-9-]{0,63}$`) → 201 `{ "key", "token" }` — full token shown once, only its SHA256 stored. Multiple active keys per app allowed (zero-downtime rotation: add new → redeploy app → revoke old).
- `DELETE /apikeys/{id}` → 204, sets `revoked_at` (idempotent).

### `/queries` (session)

Saved filter sets: `params` = `{ app?, levels? (string[]), q?, request_id? }` — no time fields.

- `GET /queries` → `{ "queries": [ { "id", "name", "params", "created_at" } ] }` ordered by name
- `POST /queries` body `{ "name", "params" }` → 201 `{ "query" }`; duplicate name → 409
- `DELETE /queries/{id}` → 204; referenced by alert rules → 409 "delete dependent alert rules first"

### `/alerts` (session + admin)

Rules reference a saved query (FK `ON DELETE RESTRICT`) and fire a webhook when the query
matches ≥ `threshold` entries in the last `window_minutes`. A 60s evaluator goroutine skips
rules fired within their window (re-arm after a full window); `last_fired_at` is set only on
a 2xx webhook response, so failures retry next tick. Payload:
`{ alert, query, count, threshold, window_minutes, since, until, entries[≤5 newest] }`,
optionally with a custom auth header (`webhook_header: webhook_secret` — secret is write-only,
never returned).

- `GET /alerts` → `{ "alerts": [ { "id", "name", "saved_query_id", "query_name", "threshold", "window_minutes", "webhook_url", "webhook_header", "enabled", "last_fired_at", "created_at" } ] }`
- `POST /alerts` body `{ "name", "saved_query_id", "threshold", "window_minutes", "webhook_url", "webhook_header"?, "webhook_secret"? }` → 201 `{ "alert" }`
- `PATCH /alerts/{id}` body `{ "enabled" }` → 200 `{ "alert" }`
- `DELETE /alerts/{id}` → 204 idempotent

### `GET /apps`

Response: `{ "apps": [ { "name", "count", "last_seen" } ] }` — for the filter rail.

### `GET /health`, `GET /ready`

`{ "status": "ok" }` / readiness pings the DB.

## Conventions

- API modules follow the Nuage pattern: each `modules/<name>/` has `router.go` (`RegisterRoutes`),
  `handler.go`, `service.go`, `types.go`.
- GORM models live in `apps/api/schemas/`; migration in `schemas/migrate.go`.
- Client uses Svelte 5 runes only (`$state`, `$props`, `$derived`, `$effect`), TypeScript strict.
- All client API calls go through `src/lib/backend.ts`.

## Gotchas

- The API Dockerfile context is the repo root (it copies `apps/api/`). The client Dockerfile
  context is `apps/client/`. Both have `.dockerignore` files.
- Ingest auth is per-app API keys (created on the dashboard's Keys page, admin only). The legacy
  `INGEST_TOKEN` still works if set; empty (the default) disables it — with no keys and no legacy
  token, every `/ingest` is rejected.
- `docker compose up` alone publishes **no** host ports (production shape). Local dev needs
  `-f docker-compose.yml -f docker-compose.dev.yml`, which binds 3000/4010/5432 on 127.0.0.1.
- Live tail polls `GET /logs` every 2.5s and merges entries whose `id` exceeds the current max
  (capped at 2000 rows client-side). It relies on `id` monotonicity, not `created_at`, so
  out-of-order client timestamps still tail correctly. The histogram refreshes every 4th poll.
- In-flight request races on the dashboard are guarded by generation counters — stale load/poll
  responses are discarded, not merged.
- Default ports: API `4010`, client `3000` — chosen to not clash with Nuage (`4000`/`3000`,
  different host).
- Full-text uses the `simple` dictionary (no stemming/stopwords) for predictable, language-agnostic
  matching across app log lines.
- The `iconify-icon` script is vendored in `static/vendor/` (no CDN at runtime), but it still
  fetches icon *data* from `api.iconify.design` — the CSP `connect-src` must keep allowing that
  origin or every icon breaks.
- CSP is configured in `svelte.config.js` (`kit.csp`, auto nonces); the other security headers
  live in `src/hooks.server.ts`. The Go API sets its own headers for `/api/*` (the prod path).

## Collector sidecar

`apps/collector` (stdlib-only Go) tails every Docker container on the host via
`/var/run/docker.sock` and ships lines to `/ingest` — zero code change for apps that only
write stdout/stderr. Opt-in via compose profile: set `COMPOSE_PROFILES=collector` in the
deploy env. It ships many apps, so it needs the **legacy unscoped `INGEST_TOKEN`** (per-app
keys won't work). Container labels: `journal.ignore=true` to skip, `journal.app=<name>` to
override the app name. On restart it resumes from "now" (small loss accepted). See
`apps/collector/README.md`.

## Later drawer (see ROADMAP.md §3 for triggers)

- **Partitioning**: monthly partitions + drop-partition retention once `log_entries` reaches
  ~10GB (the `RETENTION_DAYS` delete job covers current volume).
- **OTLP `/v1/logs`** when a real OTel-instrumented app needs it.
- **ClickHouse/VictoriaLogs migration path**: once volume outgrows Postgres (~100M rows), keep
  the same HTTP contract so the client and shippers don't change.
