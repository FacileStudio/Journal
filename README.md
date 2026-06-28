# Journal

Centralized logging service for the Facile Suite. Facile apps ship structured log entries
to Journal over HTTP; a SvelteKit dashboard lets you search, filter, and live-tail them.

## Architecture

Single public endpoint: the SvelteKit client serves the dashboard and proxies `/api/*`
requests to the Go API internally. Postgres is an internal Docker service with hardcoded
credentials.

```
Facile apps ──POST /ingest (Bearer INGEST_TOKEN)──▶ Go API (:4010) ──▶ Postgres
Browser ──login──▶ SvelteKit (:3000) ──/api/* (Bearer session)──▶ Go API (:4010)
```

The dashboard is behind email/password login (mirrors the Nuage pattern: Argon2id passwords,
DB-backed sessions). `/logs` and `/apps` require a valid session; `/ingest` keeps its separate
machine token. The first account created becomes admin — set `ALLOW_REGISTRATION=false` to lock
sign-ups once your accounts exist (the first account is always allowed, so you can't lock
yourself out).

## Stack

- `apps/api`: Go, Chi, GORM, PostgreSQL (full-text search via `tsvector` + GIN)
- `apps/client`: SvelteKit 5, Tailwind CSS 4, Bun
- `docker-compose.yml`: PostgreSQL, API, and client services

## Quick start

### Docker

```sh
cp .env.example .env
docker compose up --build
```

Open `http://localhost:3000` and create the first account — it becomes the admin.

### Local development

1. Start PostgreSQL:

```sh
docker compose up journal-db -d
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

```sh
curl -X POST http://localhost:4010/ingest \
  -H "Authorization: Bearer change-me" \
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
  -H "Authorization: Bearer change-me" \
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
const JOURNAL_TOKEN = process.env.JOURNAL_TOKEN ?? 'change-me';

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
| `INGEST_TOKEN` | Bearer token required to POST `/ingest` | `change-me` |
| `ALLOW_REGISTRATION` | `false` locks dashboard sign-ups (first account always allowed) | `true` |
| `ALLOWED_ORIGINS` / `DOMAINS` | Allowed frontend origins for CORS | — |
| `LOG_LEVEL` | `debug`, `info`, `warn`, or `error` | `info` |
| `DATABASE_URL` | Postgres connection string | internal Docker default |
| `PORT` | API listen port | `4010` |

See [`.env.example`](.env.example) for a production-ready template.
