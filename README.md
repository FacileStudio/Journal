# Journal

Centralized logging service for the Facile Suite. Facile apps ship structured log entries
to Journal over HTTP; a SvelteKit dashboard lets you search, filter, and live-tail them.

## Architecture

The SvelteKit client serves the dashboard; in production Traefik routes `/api/*` directly to
the Go API (the client's reverse proxy covers local dev). Postgres is an internal Docker
service with hardcoded credentials and no published ports.

```
Facile apps ──POST /ingest (Bearer per-app key)──▶ Go API (:4010) ──▶ Postgres
Browser ──login──▶ SvelteKit (:3000) ──/api/* (Bearer session)──▶ Go API (:4010)
```

The dashboard is behind email/password login (mirrors the Nuage pattern: Argon2id passwords,
DB-backed sessions). `/logs` and `/apps` require a valid session; `/ingest` takes a per-app
API key created on the dashboard's Keys page (admin only) — or the legacy shared
`INGEST_TOKEN` if you set one. The first account created becomes admin — set
`ALLOW_REGISTRATION=false` to lock sign-ups once your accounts exist (the first account is
always allowed, so you can't lock yourself out). Log entries older than `RETENTION_DAYS`
(default 90) are deleted hourly.

## Stack

- `apps/api`: Go, Chi, GORM, PostgreSQL (full-text search via `tsvector` + GIN)
- `apps/client`: SvelteKit 5, Tailwind CSS 4, Bun
- `docker-compose.yml`: PostgreSQL, API, and client services

## Quick start

### Docker

```sh
cp .env.example .env
docker compose -f docker-compose.yml -f docker-compose.dev.yml up --build
```

Open `http://localhost:3000` and create the first account — it becomes the admin. Then create
a per-app API key on the Keys page for each app that will ship logs. (Plain
`docker compose up` is the production shape: no host ports published.)

### Local development

1. Start PostgreSQL:

```sh
docker compose -f docker-compose.yml -f docker-compose.dev.yml up journal-db -d
```

2. Start the API (port 4010):

```sh
cd apps/api
cp .env.example .env
go run .
```

3. Start the client (port 5173) in another terminal:

```sh
cd apps/client
bun install
bun run dev
```

## Shipping a test log

Create a key on the dashboard's Keys page (or set the legacy `INGEST_TOKEN`), then:

```sh
curl -X POST http://localhost:4010/ingest \
  -H "Authorization: Bearer journal_nuage_..." \
  -H "Content-Type: application/json" \
  -d '{
    "app": "nuage",
    "level": "error",
    "message": "failed to upload file",
    "meta": { "file_id": 42, "size": 10485760 }
  }'
```

Batch ingest:

```sh
curl -X POST http://localhost:4010/ingest \
  -H "Authorization: Bearer journal_opus_..." \
  -H "Content-Type: application/json" \
  -d '{ "entries": [
    { "app": "opus", "level": "info", "message": "task created" },
    { "app": "opus", "level": "warn", "message": "due date in the past" }
  ] }'
```

## Shipping logs from a Facile app

A tiny client any Facile app can drop in:

```ts
const JOURNAL_URL = process.env.JOURNAL_URL ?? 'http://localhost:4010';
const JOURNAL_TOKEN = process.env.JOURNAL_TOKEN ?? '';

export async function shipLog(
	level: 'debug' | 'info' | 'warn' | 'error',
	message: string,
	meta?: Record<string, unknown>
) {
	await fetch(`${JOURNAL_URL}/ingest`, {
		method: 'POST',
		headers: {
			'Content-Type': 'application/json',
			Authorization: `Bearer ${JOURNAL_TOKEN}`
		},
		body: JSON.stringify({ app: 'my-app', level, message, ts: new Date().toISOString(), meta })
	}).catch(() => {});
}

shipLog('info', 'app started', { version: '1.2.3' });
```

The `.catch(() => {})` keeps logging best-effort — Journal being down should never take an
app down with it.

## Configuration

| Variable | Description | Default |
|---|---|---|
| `ORIGIN` | Public URL of the SvelteKit app (CSRF) | `http://localhost:3000` |
| `INGEST_TOKEN` | Legacy shared ingest token; empty disables it (per-app keys are primary) | — (empty) |
| `RETENTION_DAYS` | Delete log entries older than N days; `0` keeps forever | `90` |
| `ALLOW_REGISTRATION` | `false` locks dashboard sign-ups (first account always allowed) | `true` |
| `ALLOWED_ORIGINS` / `DOMAINS` | Allowed frontend origins for CORS | — |
| `LOG_LEVEL` | `debug`, `info`, `warn`, or `error` | `info` |
| `DATABASE_URL` | Postgres connection string | internal Docker default |
| `PORT` | API listen port | `4010` |

See [`.env.example`](.env.example) for a production-ready template.
