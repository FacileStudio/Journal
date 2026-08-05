# Journal — API

Every HTTP route the Go binary registers, grouped by module, as declared in
`apps/api/main.go` and each module's `router.go`.

Application routes live under `/api`: `/ingest` is served at `/api/ingest`, `/logs` at
`/api/logs`. `/health` and `/ready` answer at both the root and under `/api`. Anything else
falls through to the dashboard's index document — including a `POST`, which is why a shipper
pointed at the bare host sees `200` for every dropped line.

Credentials are always `Authorization: Bearer <token>`. The scheme is required and matched
case-insensitively.

| Auth column | Meaning |
|---|---|
| public | No credential |
| session | Dashboard session token |
| admin | Session token whose user has `is_admin` |
| ingest | Per-app API key, or the legacy `INGEST_TOKEN` |

## Health

| Method | Path | Auth |
|---|---|---|
| GET | `/health`, `/api/health` | public |
| GET | `/ready`, `/api/ready` | public |

`/health` returns `200 {"status":"ok"}` without touching a dependency. `/ready` pings
Postgres with a 2-second timeout and returns `200 {"status":"ready"}` or
`503 {"status":"not_ready"}`.

## Auth

| Method | Path | Auth |
|---|---|---|
| GET | `/api/auth/config` | public |
| POST | `/api/auth/register` | public |
| POST | `/api/auth/login` | public |
| POST | `/api/auth/logout` | session |
| GET | `/api/auth/me` | session |

`GET /api/auth/config` returns `{"sso_only":false,"oidc_enabled":false,"allow_registration":bool}`.
Journal has no OIDC client, so the first two fields are constants — the shape exists so the
login screen matches the rest of the suite.

`POST /api/auth/register` takes `{"email","name","password"}` and returns `201 {"token","user"}`.
The email must parse, the password must be at least 12 characters, a duplicate email is
`409`, and registration while disabled is `403`. The count of existing users is taken inside
a transaction holding a `pg_advisory_xact_lock`, so the first account reliably becomes admin
and `ALLOW_REGISTRATION=false` cannot lock an empty instance out.

`POST /api/auth/login` takes `{"email","password"}` and returns `200 {"token","user"}`, or
`401 "invalid email or password"` for both a wrong password and an unknown account. It also
opportunistically deletes expired session rows.

`user` is `{"id","email","name","is_admin","created_at"}`. Session tokens last 30 days.

## Ingest

| Method | Path | Auth |
|---|---|---|
| POST | `/api/ingest` | ingest |

Accepts a single entry or a batch:

```json
{ "app": "nuage", "level": "error", "message": "upload failed", "meta": { "file_id": 42 } }
```

```json
{ "entries": [ { "app": "opus", "message": "task created" } ] }
```

Entry fields:

| Field | Rules |
|---|---|
| `app` | Required with the legacy token. With a per-app key it may be omitted (filled from the key) or must equal the key's app, else `400` |
| `level` | `debug`, `info`, `warn`, or `error`. Defaults to `info`. Anything else is `400` |
| `message` | Required. Truncated at 64 KiB on a rune boundary with `" [truncated]"` appended |
| `ts` | Optional RFC3339, stored as `created_at`. More than 5 minutes in the future is replaced by server time |
| `meta` | Optional object, stored as `jsonb` |

Limits: at most 1000 entries per batch, otherwise `400`. `Content-Encoding: gzip` is
accepted, with an 8 MB cap on the compressed body and 32 MB on the decompressed one. Both
caps also apply to uncompressed bodies, which is the practical trap here — **a plain JSON
body above 8 MB is rejected**, so a shipper batching large `meta` payloads must split them.
Rate limiting is 600/min per token hash, answered as `429` with `Retry-After: 60`.

Response `201 {"ingested": n}`. Entries are written in batches of 500.

## Logs

| Method | Path | Auth |
|---|---|---|
| GET | `/api/logs` | session |
| GET | `/api/logs/histogram` | session |
| GET | `/api/logs/{id}/context` | session |
| GET | `/api/apps` | session |

`GET /api/logs` query parameters:

| Parameter | Meaning |
|---|---|
| `app` | Exact match on the source app |
| `level` | Repeatable or comma-separated, matched with `IN` |
| `q` | Full text, `websearch_to_tsquery('simple', q)` against the generated `search` column |
| `request_id` | Exact match on `meta->>'request_id'` |
| `since`, `until` | RFC3339 bounds on `created_at` |
| `limit` | Default 100, clamped to 1000 |
| `before_ts`, `before_id` | Keyset cursor, both or neither. Predicate `(created_at, id) < (?, ?)` |

Supplying only one half of the cursor is `400`. Results are ordered `created_at desc, id
desc`. The response is `{"entries":[…],"next_before":{"ts","id"}|null}`; `next_before` is
non-null only when the page came back full, and its `ts` is RFC3339Nano.

`GET /api/logs/histogram` takes the same filters minus the cursor and limit. `until`
defaults to now, `since` to 24 hours before `until`, and `until` must be after `since` or it
is `400`. The server picks the smallest bucket from 1m, 5m, 15m, 1h, 6h, or 1d that yields
at most 90 buckets. The response is
`{"bucket_seconds":n,"buckets":[{"ts","counts":{"error":n}}]}`, with empty buckets and
zero-count levels omitted — the client fills the gaps.

`GET /api/logs/{id}/context?before=50&after=50` returns the surrounding stream ignoring all
filters. Each side defaults to 50 and is capped at 200; an unknown id is `404`. The response
is `{"entries":[…],"anchor_id":n}` ordered newest first with the anchor included.

`GET /api/apps` returns `{"apps":[{"name","count","last_seen"}]}` grouped by app and ordered
by most recently seen, which is what the filter rail reads.

## API keys

| Method | Path | Auth |
|---|---|---|
| GET | `/api/apikeys` | admin |
| POST | `/api/apikeys` | admin |
| DELETE | `/api/apikeys/{id}` | admin |

`POST` takes `{"app"}`, which must match `^[a-z0-9][a-z0-9-]{0,63}$`, and returns
`201 {"key","token"}`. The `token` is the only time the full credential is visible; the row
stores a display `prefix` and the SHA-256 hash. Several active keys per app are allowed,
which is what makes zero-downtime rotation possible: mint the new key, redeploy the shipper,
then revoke the old one.

`DELETE` returns `204` and sets `revoked_at`. It is idempotent, and revoking an unknown id
is `404`.

## Saved queries

| Method | Path | Auth |
|---|---|---|
| GET | `/api/queries` | session |
| POST | `/api/queries` | session |
| DELETE | `/api/queries/{id}` | session |

A saved query is a named filter set: `params` is `{"app","levels","q","request_id"}` with no
time bounds, since a saved time range would go stale. `levels` must be a subset of the four
valid levels or the request is `400`.

`GET` returns `{"queries":[{"id","name","params","created_at"}]}` ordered by name. `POST`
takes `{"name","params"}` and returns `201 {"query"}`; a duplicate name is `409`. `DELETE`
returns `204`, or `409 "delete dependent alert rules first"` when an alert rule references
it — checked explicitly and again through the `ON DELETE RESTRICT` foreign key.

## Alerts

| Method | Path | Auth |
|---|---|---|
| GET | `/api/alerts` | admin |
| POST | `/api/alerts` | admin |
| PATCH | `/api/alerts/{id}` | admin |
| DELETE | `/api/alerts/{id}` | admin |

A rule fires when its saved query matches at least `threshold` entries within the last
`window_minutes`. `POST` takes
`{"name","saved_query_id","threshold","window_minutes","webhook_url","webhook_header","webhook_secret"}`.
Validation: `name` non-empty, `threshold` at least 1, `window_minutes` between 1 and 1440,
`webhook_url` a parseable `http` or `https` URL that does not literally name a private,
loopback, or metadata IP. An unknown `saved_query_id` is `404`.

`webhook_secret` is write-only and never returned. When both header fields are set, delivery
sends `webhook_header: webhook_secret`.

`PATCH` takes `{"enabled"}` and returns `200 {"alert"}`; omitting the field is `400`.
`DELETE` returns `204`.

The evaluator runs every minute, skips rules that fired inside their own window, and writes
`last_fired_at` only on a 2xx webhook response, so a failing endpoint is retried rather than
silently marked delivered. The payload is:

```json
{
  "alert": "errors spiking",
  "query": "nuage errors",
  "count": 42,
  "threshold": 10,
  "window_minutes": 15,
  "since": "2026-08-05T10:00:00Z",
  "until": "2026-08-05T10:15:00Z",
  "entries": []
}
```

`entries` carries at most the five newest matching log entries. Delivery refuses redirects,
and unless the target host is listed in `WEBHOOK_ALLOWED_HOSTS` the dialer blocks loopback,
link-local, multicast, unspecified, and private addresses.

## Errors and rate limits

Errors come from `tronc/errors` and are written by `tronc/httpjson`, so every failure is
JSON with the status the error type maps to: `400` invalid, `401` unauthorized, `403`
forbidden, `404` not found, `409` conflict, `429` rate limited, `500` internal.

| Group | Limit | Key |
|---|---|---|
| `/api/auth/register`, `/api/auth/login` | 20/min | client IP and endpoint |
| Other session routes | 300/min | client IP |
| `/api/ingest` | 600/min | SHA-256 of the bearer token |

Health and readiness are exempt.
