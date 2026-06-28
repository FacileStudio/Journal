# Journal

Centralized logging service for the Facile Suite. Facile apps POST structured log entries to
a Go API; a SvelteKit dashboard searches, filters, and live-tails them. Self-hosted,
Docker-deployed.

## Tech Stack

| Layer | Tech |
|-------|------|
| API | Go 1.24, Chi router, GORM, PostgreSQL 16 (full-text search via `tsvector` + GIN) |
| Client | SvelteKit 5 (Svelte 5 runes), Tailwind CSS 4, Bun, adapter-node |
| Auth | Static Bearer token (`INGEST_TOKEN`) on `/ingest` only |
| Infra | Docker Compose, Traefik (production), Dokploy |

## Key Commands

### Docker (full stack)

```sh
cp .env.example .env
docker compose up --build          # client on localhost:3000, api on localhost:4010
```

### Local Development

```sh
# 1. Start Postgres
docker compose up journal-db -d

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
  docker-compose.yml               # db, api, client
  docker-compose.override.yml      # exposes ports for local dev
  .env.example                     # production env template
  apps/
    api/
      main.go                      # entrypoint, router + middleware stack, route registration
      internal/
        env/                       # config loading from env vars
        database/                  # GORM Postgres connection
        httpjson/                  # JSON decode/encode + error helpers
        errors/                    # typed errors -> HTTP status mapping
        logger/                    # structured slog logging
        middleware/                # CORS, security headers, request logging, ingest token
      schemas/                     # GORM models + Migrate (AutoMigrate + raw SQL indexes)
      modules/
        ingest/                    # POST /ingest (single + batch), token-protected
        logs/                      # GET /logs (search/filter/cursor), GET /apps
    client/
      src/
        lib/backend.ts             # typed API client (listLogs, listApps)
        routes/
          +page.svelte             # logs dashboard (filters, search, live tail, expand)
          api/[...path]/           # reverse proxy to Go API
      static/                      # favicon, logo, fonts
```

## Architecture

```
Facile apps ──POST /ingest──▶ Go API (:4010) ──▶ Postgres
Browser ──▶ SvelteKit (:3000) ──/api/*──▶ Go API (:4010)
```

The SvelteKit client is the only public surface. It reverse-proxies `/api/*` to the Go API.
Postgres is internal with hardcoded credentials. In production Traefik strips the `/api`
prefix before the API and routes the rest to the client.

## Environment Variables

| Variable | Description | Default |
|---|---|---|
| `DATABASE_URL` | Postgres connection string | `postgres://journal:journal@localhost:5432/journal?sslmode=disable` |
| `PORT` | API listen port | `4010` |
| `LOG_LEVEL` | `debug`, `info`, `warn`, `error` | `info` |
| `INGEST_TOKEN` | Bearer token required to POST `/ingest` | — (empty rejects all ingest) |
| `ALLOWED_ORIGINS` / `DOMAINS` | Comma-separated CORS origins | — |
| `ORIGIN` | Public client URL (CSRF) | `http://localhost:3000` |

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

Extra indexes: GIN on `search`, composite `(app, created_at DESC)`. `schemas.Migrate` runs
`AutoMigrate` then raw SQL for the generated column + indexes (GORM can't express a generated
`tsvector` column or a `DESC` composite index).

## API Contract

### `POST /ingest` (Bearer `INGEST_TOKEN`)

Single entry **or** batch. Entry fields: `app` (required), `level` (optional, default `info`),
`message` (required), `ts` (optional RFC3339 -> `created_at`), `meta` (optional object).

```jsonc
{ "app": "nuage", "level": "error", "message": "boom", "meta": { "k": "v" } }
// or
{ "entries": [ { "app": "opus", "message": "task created" } ] }
```

Response `201`: `{ "ingested": <n> }`.

### `GET /logs`

Query params: `app`, `level` (repeatable or CSV), `q` (full-text via
`websearch_to_tsquery('simple', q)`), `since`/`until` (RFC3339 on `created_at`), `limit`
(default 100, max 1000), `before` (id cursor — returns rows with `id < before`). Ordered
`created_at desc, id desc`.

Response: `{ "entries": [...], "next_before": <id|null> }`.

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
  context is `apps/client/`.
- `INGEST_TOKEN` empty -> every `/ingest` is rejected. Set it everywhere, including dev.
- Live tail polls `GET /logs` every 2.5s and merges entries whose `id` exceeds the current max.
  It relies on `id` monotonicity, not `created_at`, so out-of-order client timestamps still tail
  correctly.
- Default ports: API `4010`, client `3000` — chosen to not clash with Nuage (`4000`/`3000`,
  different host).
- Full-text uses the `simple` dictionary (no stemming/stopwords) for predictable, language-agnostic
  matching across app log lines.

## Pass 2 (not in MVP)

- **Per-app API tokens**: replace the single `INGEST_TOKEN` with per-app tokens (table +
  middleware lookup) so a leaked token scopes to one app and can be rotated independently.
- **Alerting rules**: stored rules (app + level + text/rate threshold) evaluated on ingest,
  firing to Nook webhooks / email when matched.
- **Docker log auto-collector**: a sidecar tailing container stdout/json-file logs and shipping
  to `/ingest`, so apps that only `console.log` are captured with zero code change.
- **Retention / partitioning**: time-based partitioning of `log_entries` (monthly) + a retention
  job dropping old partitions; keeps the hot set small and deletes O(1).
- **ClickHouse migration path**: once volume outgrows Postgres, mirror ingest to ClickHouse
  (MergeTree, `app`/`created_at` ordering) for cheap columnar scans; keep the same HTTP contract
  so the client and shippers don't change.
