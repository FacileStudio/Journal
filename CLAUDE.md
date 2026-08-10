# Journal

Centralized logging service for the Facile Suite. Facile apps POST structured log entries to
a Go API; a SvelteKit dashboard searches, filters, and live-tails them. Self-hosted,
Docker-deployed.

## Tech Stack

| Layer | Tech |
|-------|------|
| API | Go 1.24, Chi router, GORM, PostgreSQL 16 (full-text search via `tsvector` + GIN) |
| Client | SvelteKit 5 (Svelte 5 runes), Tailwind CSS 4, Bun, adapter-static (served by the Go binary) |
| Design system | [muse](https://github.com/FacileStudio/muse) (`@facile/muse`, pinned to `#v0.4.0`) — owns the palette, Goga, the `fc-*` tokens, dark mode and every primitive |
| Auth | Dashboard: email/password accounts, DB-backed sessions (Argon2id, 30-day token). Ingest: per-app API keys (SHA256-hashed, admin-managed) in two kinds — **secret** for servers, **public** for browsers — with optional legacy static `INGEST_TOKEN`. Mirrors the Nuage pattern. |
| Infra | Docker Compose, Traefik (production), Dokploy |

## Key Commands

### Docker (full stack)

```sh
cp .env.example .env
docker compose up --build                                          # production: no host ports published
docker compose -f docker-compose.yml -f docker-compose.dev.yml up  # dev: 127.0.0.1:4010/5432
```

### Local Development

```sh
# 1. Start Postgres
docker compose -f docker-compose.yml -f docker-compose.dev.yml up journal-db -d

# 2. API (port 4010)
cd apps/api
cp .env.example .env
go run .

# 3. Client (port 5173) — Vite proxies /api to the API above
cd apps/client
bun install
bun run dev
```

### Browser SDK

```sh
cd sdk/browser
bun install && bun test && bun run build   # dist/ is committed: it is what consumers install
```

### Ingest a test log

```sh
curl -X POST http://localhost:4010/api/ingest \
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
  Dockerfile                       # one image: bun builds the client, go builds the API, distroless runs it
  docker-compose.yml               # db + one app service — production shape, no host ports
  docker-compose.dev.yml           # opt-in (-f) local dev: publishes ports on 127.0.0.1
  .env.example                     # production env template
  apps/
    api/
      main.go                      # entrypoint, router + middleware stack, retention job, route registration under /api, SPA catch-all
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
        auth/                      # /api/auth/{config,register,login,logout,me} — sessions
        ingest/                    # POST /api/ingest (single + batch, gzip), per-app key or legacy token
                                   # + POST /api/ingest/browser (public key, origin-checked, quotaed)
        logs/                      # GET /api/logs, /api/logs/histogram, /api/logs/{id}/context, GET /api/apps — session-protected
        apikeys/                   # /api/apikeys CRUD — session + admin only
        queries/                   # /api/queries CRUD (saved filter sets) — session-protected
        alerts/                    # /api/alerts CRUD + 60s webhook evaluator — session + admin only
    collector/                     # optional sidecar: tails all Docker containers via docker.sock, ships to /api/ingest
    client/
      src/
        app.css                    # `@import '@facile/muse/styles'` + `@source` + the suite-name alias block. Nothing else.
        lib/backend.ts             # typed API client (auth, logs, histogram, context, api keys, queries, alerts)
        lib/auth.ts                # localStorage session token (journal.token)
        lib/theme.svelte.ts        # system/light/dark, toggles BOTH .dark and .light on <html>
        lib/format.ts              # formatTime / formatClock / formatDate / formatRelative / toLocalInput
        lib/levels.ts              # level -> muse tone, chart fill, chip class, volume-strip weight
        lib/histogram.ts           # sparse buckets -> dense bars, stacked pixel geometry (pure, testable)
        lib/components/            # LogTable, LogHistogram, LogContextDrawer, LevelBadge, PageHeader
        routes/
          +layout.svelte           # app.css, iconify-icon registration, stored theme, one <Toaster>
          +layout.ts               # prerender = false, ssr = false — the whole dashboard is client-rendered
          login/+page.svelte       # canonical suite split-screen sign in / register
          (app)/+layout.svelte     # the shell: SideBar + Topbar + MobileNav, auth guard, user context
          (app)/+page.svelte       # Overview: stat cards, volume strip, level donut, busiest apps, recent errors
          (app)/logs/+page.svelte  # explorer: filters (URL-synced), histogram drill-down, live tail, pivots, context drawer
          (app)/apps/+page.svelte  # sources with counts and last-seen
          (app)/queries/+page.svelte # saved filter sets
          (app)/alerts/+page.svelte  # alert rules (admin only)
          (app)/settings/          # +layout (Tabs) + Profile / appearance / api (keys, admin) / advanced
      static/                      # favicon, logo — fonts and iconify come from the muse package
  sdk/
    journal/                       # Go SDK: batching client + slog tee handler (stdlib-only, go-gettable)
    browser/                       # @facile/journal: dependency-free browser SDK + SvelteKit handleError
                                   # dist/ is committed — it is what github:FacileStudio/Journal#ts installs
```

## Architecture

```
Facile apps ──POST /api/ingest──▶ Go API (:4010) ──▶ Postgres
Browser ──▶ same Go API (:4010): /api/* is the API, everything else is the static dashboard
```

One container. The Go binary registers every application module inside
`router.Route("/api", …)` and mounts tronc's `spa.Handler` on `/*` as the last route, so it
serves both halves; Traefik has a **single** router, ``Host(`journal.facile.studio`)``, with
no `PathPrefix` and no stripprefix middleware.

Public URLs did not change when this replaced the two-container split: Traefik used to strip
`/api` before forwarding, so `/api/ingest` reached the API as `/ingest`. Now the prefix is
part of the route itself and `/api/ingest` stays `/api/ingest` — the nine suite apps and the
collector shipping to `JOURNAL_URL=https://journal.facile.studio/api` are unaffected.

`/health` and `/ready` are explicit chi routes at the root, so they win over the SPA
catch-all and keep their URLs. The Go API's own middleware (CORS, security headers, rate
limits) is the only browser-facing perimeter — there is no SvelteKit server left. Postgres is
internal with hardcoded credentials and no published ports.

## Environment Variables

**Development configuration comes from [Casier](https://casier.facile.studio), not from a
`.env`.** `casier.yml` pins this repo to the `journal` project and its `dev` environment, so
`mise run dev` wraps the API in `casier run` and the process starts with `ORIGIN`,
`INGEST_TOKEN`, `LOG_LEVEL` and `ALLOWED_ORIGINS` already in its environment. Nothing has to
exist on disk; a local `.env` still works and simply wins nothing, because the API reads the
process environment either way.

`casier run` is network-first with a last-known-good cache, so a Casier outage falls back to
the previously fetched values and *says so on stderr* — it never starts the API with an empty
environment. `mise run dev-offline` forces the cached path. Add or change a value with
`casier secrets set -p journal -e dev KEY value`, or push a whole file with
`casier sync push -p journal -e dev -f .env`.

Deployment is unchanged: production values are injected by Dokploy. Casier covers the
developer loop, not the bootstrap tier.


| Variable | Description | Default |
|---|---|---|
| `DATABASE_URL` | Postgres connection string | **required** — the API exits 1 without it |
| `APP_ENV` | `development`, `staging`, `production`. Never gates security behaviour | `development` |
| `PORT` | API listen port | `4010` |
| `LOG_LEVEL` | `debug`, `info`, `warn`, `error` | `info` |
| `INGEST_TOKEN` | Legacy shared ingest token (unscoped). Empty disables it — per-app API keys are the primary ingest auth | — (empty) |
| `RETENTION_DAYS` | Delete log entries older than N days (hourly job); `0` keeps forever | `90` |
| `ALLOW_REGISTRATION` | `false` locks dashboard sign-ups (first account always allowed) | `true` |
| `OIDC_ISSUER` | Enables single sign-on through porte; makes the next four required | — |
| `OIDC_CLIENT_ID` / `OIDC_CLIENT_SECRET` | Provider credentials | — |
| `OIDC_REDIRECT_URL` / `OIDC_SUCCESS_URL` | Callback and landing URLs | — |
| `SSO_ONLY` | `true` unregisters the password routes; needs `OIDC_ISSUER` | `false` |
| `CORS_ALLOWED_ORIGINS` | Comma-separated CORS origins, read through `tronc/env`. `ALLOWED_ORIGINS` and `DOMAINS` remain accepted fallbacks | — (unset denies every cross-origin caller) |
| `WEBHOOK_ALLOWED_HOSTS` | Comma-separated hostnames allowed as internal alert webhook targets | — |
| `CLIENT_DIR` | Directory holding the built dashboard; the SPA route is skipped if it has no `index.html` | `./client` |

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

Table `api_keys`: `id` PK, `app` text, `kind` text (`secret`|`public`, default `secret`),
`prefix` text (display only), `key_hash` text unique (SHA256 hex of the full token),
`allowed_origins` jsonb (public keys only), `daily_quota` int (`0` = unlimited, legal on secret
keys only), `created_at`, `revoked_at` nullable.

Table `api_key_usage`: `(api_key_id, day)` PK, `count` bigint. One row per key per UTC day,
incremented by the browser endpoint *before* it writes. FK cascades on key delete.

## API Contract

Every path below is relative to the `/api` mount: `/ingest` is served at `/api/ingest`,
`/logs` at `/api/logs`, and so on. The only exceptions are `/health` and `/ready`, which stay
at the root. Anything not under `/api` falls through to the dashboard's index document.

### Auth (`/auth/*`)

Dashboard accounts, through **porte** — the suite's shared authentication kit, which Journal
is the first app to adopt. `porte/oidc` owns the middleware, `/auth/config`, `/auth/logout`
and the whole OIDC flow; `porte/pg` owns `porte_sessions`, `porte_identities` and
`porte_login_codes`. Journal keeps its own `users` table and implements `porte.UserStore` over
it, because `is_admin` and first-account-becomes-admin are product rules porte has no opinion
about.

Sessions are DB-backed: a random 32-byte token, stored SHA256-hashed in `porte_sessions`, 30-day
absolute TTL plus a 7-day idle window on the cookie transport only. Two transports, one row —
a password login returns the token and the client keeps it in `localStorage` (`journal.token`)
and sends `Authorization: Bearer <token>`; the SSO callback puts it in an HttpOnly
`__Host-session` cookie instead and no token ever reaches a URL. A cookie-authenticated write
must carry `X-Facile-CSRF` with any non-empty value. Passwords are `porte/local`'s Argon2id — same parameters as before the move, so existing
hashes keep verifying — and the first account created becomes admin (guarded by `pg_advisory_xact_lock`, in the register transaction
and in the OIDC upsert alike). Login opportunistically deletes expired session rows.

- `GET /auth/config` → `{ sso_only, oidc_enabled, allow_registration }` (porte serves it; `allow_registration` rides in through `Deps.ConfigExtra`)
- `GET /auth/oidc` → starts SSO. Registered only with an `OIDC_ISSUER`. `?flow=cli` ends on a one-time code instead of a cookie.
- `POST /auth/register` → `{ token, user }` (201). Body `{ email, name?, password }`, password ≥ 12 chars. Locked once accounts exist if `ALLOW_REGISTRATION=false` (first account always allowed). Duplicate email → 409.
- `POST /auth/login` → `{ token, user }`. Body `{ email, password }`.
- `POST /auth/logout` (session) → porte deletes the session and clears the cookie, `{ logged_out: true }`.
- `GET /auth/me` (session) → `{ user }` (includes `is_admin`). porte authenticates, then `middleware.RequireAuth` hydrates email + `is_admin` from `users` — porte carries neither.
- `SSO_ONLY=true` does not register `/auth/register` and `/auth/login` at all.

`GET /logs*` and `GET /apps` require a valid Bearer session token; `/apikeys*` additionally
requires `is_admin`. `/health` and `/ready` stay public and rate-limit exempt.

Rate limits: login/register 20/min per IP per endpoint; ingest 600/min per token hash;
browser ingest 60/min per (key, IP) under a 600/min per-key ceiling; session routes 300/min
per IP.

**The per-IP buckets hold as of tronc v0.10.0.** Until then `httpx` installed chi's `RealIP`,
which rewrote `RemoteAddr` from `X-Forwarded-For` with no check on the peer, so a rotating
header minted fresh buckets — measured here at 70 spoofed requests accepted where 10 should
have been refused. tronc's own `RealIP` believes the header only from a trusted peer and walks
the chain right to left; re-measured after the bump, the same 70 requests give 60 accepted and
10 refused. `internal/middleware/realip.go` was Journal's local (unwired) implementation and is
**deleted** — the chassis owns this now.

`TRUSTED_PROXIES` configures it. Unset means loopback plus the private ranges, which is
Traefik, so production needs no new variable; set it to Traefik's address to stop a neighbour
on the Docker network speaking for a visitor, or to `none` to key everything on the connection
address. A value that does not parse fails at boot.

The browser endpoint keeps its per-key ceiling and daily quota regardless: a bound that owes
nothing to the network layer is worth having even when the network layer is correct.

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

### `POST /ingest/browser`

The browser write path — Journal's answer to "where do client-side errors go". Authenticated
by a **public** key (`journal_pub_<app>_…`) passed as `?key=` or as a bearer token. The two
kinds never cross: a public key is rejected on `/ingest`, a secret key is rejected here.

A public key ships inside a JavaScript bundle, so it is not a secret and nothing pretends it
is. Four things bound the damage if one is abused, and they are layered on purpose:

1. **Origin allowlist** — the request's `Origin` must exactly match one of the key's stored
   origins, else 403. This is a server-side authorization check, *not* CORS: CORS only governs
   whether a script may read the response, and the request is sent either way. It stops other
   people's pages; it does not stop curl, which is what the next three are for.
2. **60/min per (key, IP)** — shapes honest traffic. Walkable around: see the gotcha below.
3. **600/min per key** — no header moves this one.
4. **Daily quota** per key, reserved before the write in one conditional upsert, so two
   concurrent requests cannot both read "just under the limit". 429 + `Retry-After` counting
   down to UTC midnight. This is the bound that actually holds.

Body: `{ release, environment, events: [{ message, level, ts, kind, stack, url, route, count,
user: { id, email }, meta }] }`. At most **20 events** and **128 KB**. There is no `app` field —
the key's app is authoritative. `meta` is scrubbed of credential-shaped keys
(`password`, `token`, `cookie`, `authorization`, …→ `"[scrubbed]"`), depth-limited to 6, arrays
cut at 50, strings at 2 KB (stack at 8 KB), and the whole object capped at 8 KB with a fallback
that keeps `origin`, `url` and half the stack. The server then stamps `source: "browser"`,
`origin` and `user_agent` **last**, so a page cannot claim any of the three.

Entries land in `log_entries` like everything else, so search, histogram, context, saved
queries and alerts all work on them unchanged.

Client: [`sdk/browser`](sdk/browser) (`@facile/journal`), consumed as
`bun add github:FacileStudio/Journal#ts`.

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

- `GET /apikeys` → `{ "keys": [ { "id", "app", "kind", "prefix", "allowed_origins", "daily_quota", "used_today", "created_at", "revoked_at" } ] }`
- `POST /apikeys` body `{ "app", "kind"?, "allowed_origins"?, "daily_quota"? }` (app `^[a-z0-9][a-z0-9-]{0,63}$`) → 201 `{ "key", "token" }` — full token shown once, only its SHA256 stored. Multiple active keys per app allowed (zero-downtime rotation: add new → redeploy app → revoke old). `kind: "public"` requires 1–8 `allowed_origins` and a `daily_quota` ≥ 1; origins are normalized on save (scheme + host lowercased, default port dropped, no path, no wildcard) so they match what a browser sends byte for byte.
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
- **muse is the component layer.** Read `muse/CHARTE.md` before writing UI: reuse a primitive
  before hand-rolling one, style with `fc-*` token utilities (never a raw hex or a stock
  Tailwind palette colour), keep the dashboard rhythm (`gap-4` inside and between cards, `p-5`
  card padding, `gap-10` between sections), and remember containers carry **no** border — the
  fill separates them. Local components exist only for domain widgets (`LogHistogram`,
  `LogTable`) and thin compositions (`PageHeader`), never for chrome, form controls or dialogs.
- Settings is reached from the sidebar user card only — never a nav row — and each section is a
  real route under `/settings` (CHARTE §14). Log out lives in settings.

## Gotchas

- There is one `Dockerfile`, at the repo root, and its context is the repo root: a bun stage
  builds the client, a Go stage builds the API, and the distroless runtime image holds the
  binary at `/api` and the client at `/client` (`spa.DefaultDir` is `./client`, and the image
  has no `WORKDIR`, so the process runs from `/`). The compose healthcheck is
  `["CMD", "/api", "healthcheck"]` — it must track the binary's path in the image. The
  runtime is `:nonroot` because the app service mounts no writable volume; adding one means
  dropping back to the root distroless variant.
- **A base URL missing `/api` fails silently.** The SPA catch-all answers *any* unmatched
  path — including a `POST` — with `200` and `index.html`, and both the collector and the Go
  SDK treat any 2xx as a successful ingest. `JOURNAL_URL` must therefore end in `/api`
  (`https://journal.facile.studio/api` publicly, `http://journal-api:4010/api` on the
  compose network); point it at the bare host and every log line is accepted, discarded, and
  never reported.
- **`/ingest/browser` must be called with `Content-Type: text/plain`.** That keeps it a CORS
  *simple* request, so no preflight is sent. An `application/json` body triggers one, and the
  preflight is answered by the app-wide CORS middleware from `CORS_ALLOWED_ORIGINS` — which
  knows nothing about a key's allowed origins — so it 403s before the route is ever reached.
  The SDK does this already; hand-rolled callers get a confusing failure.
- Ingest auth is per-app API keys (created under **Settings → API**, admin only; the page used to
  live at `/keys`). The legacy
  `INGEST_TOKEN` still works if set; empty (the default) disables it — with no keys and no legacy
  token, every `/api/ingest` is rejected.
- `docker compose up` alone publishes **no** host ports (production shape). Local dev needs
  `-f docker-compose.yml -f docker-compose.dev.yml`, which binds 4010/5432 on 127.0.0.1.
- Live tail polls `GET /logs` every 2.5s and merges entries whose `id` exceeds the current max
  (capped at 2000 rows client-side). It relies on `id` monotonicity, not `created_at`, so
  out-of-order client timestamps still tail correctly. The histogram refreshes every 4th poll.
- In-flight request races on the dashboard are guarded by generation counters — stale load/poll
  responses are discarded, not merged.
- `/logs` mirrors its filters into the query string (`app`, `level`, `q`, `request_id`, `range`,
  `since`, `until`) via `replaceState`, so a filtered view is linkable and survives a reload.
  Anything that navigates *to* the explorer (a bucket click on the Overview, an app row, a saved
  query) builds that same query string rather than poking at page state.
- The volume strip is a local component, not a muse `BarChart`: a bucket is a *control* (click to
  zoom the whole page into it) and muse's charts expose no selection. It still follows CHARTE §12
  — real pixel geometry, rounded corners on every free end, `aria-hidden` svg beside a hidden data
  table — and `debug`/`info` are painted at reduced opacity so warn and error stay legible.
- Default ports: the app listens on `4010` (chosen to not clash with Nuage's `4000`), and
  `bun run dev` still serves the client on `5173` with a Vite proxy forwarding `/api` to
  `localhost:4010`.
- Full-text uses the `simple` dictionary (no stemming/stopwords) for predictable, language-agnostic
  matching across app log lines.
- `iconify-icon` is an npm dependency registered client-side from the root layout
  (`if (browser) void import('iconify-icon')`), so it satisfies `script-src: self` without a
  vendored file. It still fetches icon *data* from `api.iconify.design` — the CSP `connect-src`
  must keep allowing that origin or every icon renders blank.
- **`optimizeDeps.exclude: ['@facile/muse']` in `vite.config.ts` is load-bearing.** muse ships
  uncompiled source including `.svelte.ts` rune modules; Vite's dev-only optimizer hands those to
  esbuild without the TypeScript transform and `vite dev` exits 1. `vite build`, `bun run check`
  and CI never run the optimizer, so nothing else catches it.
- Fonts and the whole palette come from the muse package — there is no `static/fonts`, no
  `@font-face` and no `:root`/`.dark` block in `app.css`. Do not reintroduce them.
- Theme switching must toggle **both** `.dark` and `.light` on `<html>`: muse paints dark from
  `@media (prefers-color-scheme: dark)` scoped to `:root:not(.light)`, so toggling only `.dark`
  leaves a dark-OS user unable to force light. `system` writes neither class.
- CSP is configured in `svelte.config.js` (`kit.csp`). With `adapter-static` the `auto` mode
  emits **hashes**, not nonces, into the `<meta http-equiv>` of the built `index.html`, so it
  survives being served by the Go binary. Every other security header comes from the Go
  `middleware.SecurityHeaders`, which runs on all routes including the SPA — the old
  `hooks.server.ts` is gone, since no SvelteKit server runs in production.

## Collector sidecar

`apps/collector` (stdlib-only Go) tails every Docker container on the host via
`/var/run/docker.sock` and ships lines to `/api/ingest` — zero code change for apps that only
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
