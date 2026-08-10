# Journal

Centralized logging for the Facile Suite. Apps ship structured entries over HTTP; a
SvelteKit dashboard searches, filters, and pivots them.

One Go binary serves the API under `/api` and the built dashboard on everything else.
Ingest is authenticated with per-app bearer keys, storage is PostgreSQL with full-text
search, and old entries are pruned on a schedule.

Live at [journal.facile.studio](https://journal.facile.studio).

## What it does

- Accepts single or batched log entries on `POST /api/ingest`, gzip bodies included
- Authenticates ingest with per-app API keys scoped to one `app` name, or a legacy shared token
- Collects browser errors on `POST /api/ingest/browser` — a public key restricted to an exact
  origin allowlist and bounded by a daily quota, with a scrubbed, server-stamped context
- Stores entries in PostgreSQL with a generated `tsvector` column and a GIN index
- Searches by app, level, free text, `request_id`, and time range, with keyset pagination
- Renders a level histogram and pulls the surrounding lines for any entry
- Saves named queries and turns them into threshold alerts delivered to a webhook
- Prunes entries older than `RETENTION_DAYS` every hour
- Ships a Go SDK (`sdk/journal`), a browser SDK (`sdk/browser`, `@facile/journal`) and a Docker
  log collector sidecar (`apps/collector`)

## Stack

| Layer | Tech |
|---|---|
| API | Go 1.24, Chi v5, GORM, PostgreSQL 16, [tronc](https://github.com/FacileStudio/tronc) v0.6.0 |
| Client | SvelteKit 2, Svelte 5 (runes), Tailwind CSS 4, `adapter-static`, Bun |
| Deploy | Docker Compose, single container behind Traefik |

## Quick start

```sh
cp .env.example .env
docker compose -f docker-compose.yml -f docker-compose.dev.yml up --build
```

Open <http://localhost:4010> and create the first account — it becomes the admin. Then
create a per-app API key under Settings → API for each app that will ship logs. Plain
`docker compose up` is the production shape and publishes no host ports.

Ship a test entry:

```sh
curl -X POST http://localhost:4010/api/ingest \
  -H "Authorization: Bearer journal_nuage_..." \
  -H "Content-Type: application/json" \
  -d '{"app":"nuage","level":"error","message":"failed to upload file","meta":{"file_id":42}}'
```

### Catch browser errors

Mint a **public** key under Settings → API, listing the origins your app is served from, then
two lines in SvelteKit:

```sh
bun add github:FacileStudio/Journal#ts
```

```ts
// src/hooks.client.ts
import { createJournal } from '@facile/journal';
import { handleErrorWith } from '@facile/journal/sveltekit';

const journal = createJournal({ url: 'https://journal.facile.studio/api', key: PUBLIC_JOURNAL_KEY });

journal.install();
export const handleError = handleErrorWith(journal);
```

The key belongs in the bundle — it is not a secret. What protects the instance is the origin
allowlist, a per-key rate limit and the daily quota, all enforced server-side. See
[`sdk/browser`](sdk/browser).

### Local development

```sh
docker compose -f docker-compose.yml -f docker-compose.dev.yml up journal-db -d
cd apps/api && cp .env.example .env && go run .
```

In another terminal, Vite proxies `/api` to the API on port `4010`:

```sh
cd apps/client && bun install && bun run dev
```

## Configuration

| Variable | What it does |
|---|---|
| `DATABASE_URL` | Postgres connection string, required — a missing value exits 1 |
| `PORT` | HTTP listen port, `4010` in the shipped compose file |
| `INGEST_TOKEN` | Legacy unscoped ingest token; empty disables it |
| `ALLOW_REGISTRATION` | `false` locks dashboard sign-ups; the first account is always allowed |
| `OIDC_ISSUER` | Enables single sign-on; four companion `OIDC_*` variables become required with it |
| `SSO_ONLY` | `true` retires the password routes entirely; needs `OIDC_ISSUER` |
| `RETENTION_DAYS` | Delete entries older than N days; `0` keeps forever |
| `CORS_ALLOWED_ORIGINS` | Allowed cross-origin callers; unset denies every one of them |
| `WEBHOOK_ALLOWED_HOSTS` | Hostnames alert webhooks may reach on the private network |

Full reference: [docs/configuration.md](docs/configuration.md).

## Structure

```
apps/
  api/         Go backend — modules/ (auth, ingest, logs, queries, alerts, apikeys),
               schemas/ (GORM models and migrations), internal/ (crypto, filters, middleware)
  client/      SvelteKit dashboard on the muse design system — overview, logs,
               apps, saved queries, alerts, settings
  collector/   Docker log collector sidecar, its own Go module
sdk/
  journal/     Go client and slog handler, its own Go module
  browser/     @facile/journal — browser SDK and SvelteKit handleError, dependency-free
docs/          Architecture, configuration, development, deployment, API
```

## Documentation

| Doc | What's in it |
|---|---|
| [Architecture](docs/architecture.md) | Request flow, data model, how the pieces fit |
| [Configuration](docs/configuration.md) | Every environment variable and default |
| [Development](docs/development.md) | Local setup, tests, the quality gate |
| [Deployment](docs/deployment.md) | Docker Compose, Dokploy, Traefik routing |
| [API](docs/api.md) | HTTP endpoints and payloads |

---

Part of the [Facile Suite](https://facile.studio) — self-hosted tools for creative studios
and freelancers. One login, zero cloud dependency.
