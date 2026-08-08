# Journal — Architecture

How a log entry gets from an app into Postgres, how the dashboard reads it back, and what
the alert evaluator does in between.

## Runtime topology

```
Facile apps ──POST /api/ingest (Bearer per-app key)──┐
Collector sidecar ──(Docker socket, legacy token)────┤
                                                     ▼
Internet ──▶ Traefik ──▶ Go binary (:4010) ──┬──▶ /api/*       handlers
                                              ├──▶ /health     liveness
                                              ├──▶ /ready      readiness (DB ping)
                                              └──▶ everything  SPA catch-all
                                                              │
                                                        Postgres 16
                                                        (log_entries + tsvector/GIN)
```

One container serves both halves. `apps/api/main.go` registers every module inside
`router.Route("/api", …)` and mounts `tronc/spa.Handler` on `/*` as the last route, so
Traefik needs a single router on `Host(journal.facile.studio)` — no `PathPrefix`, no strip
middleware. Postgres runs as an internal compose service with no published ports.

## Components

| Component | Path | Role |
|---|---|---|
| API | `apps/api` | Chi router, module handlers, retention job, alert evaluator |
| Client | `apps/client` | SvelteKit dashboard built with `adapter-static`, served by the Go binary |
| Collector | `apps/collector` | Optional sidecar tailing Docker containers into `/api/ingest` |
| SDK | `sdk/journal` | Go client plus a `slog` tee handler for suite apps |

`apps/collector` and `sdk/journal` are separate Go modules with their own `go.mod`, so an
app can `go get github.com/FacileStudio/Journal/sdk/journal` without pulling the API in.

## Request lifecycle

1. `tronc/httpx.NewRouter` installs request logging, real-IP resolution, and CORS built
   from `CORSAllowedOrigins`.
2. `middleware.SecurityHeaders` sets `X-Content-Type-Options`, `X-Frame-Options: DENY`,
   HSTS, `Referrer-Policy`, and `Permissions-Policy` on every response, SPA included.
3. `tronc/health.Mount` registers `/health` and `/ready` at both the root and under `/api`.
4. Rate limiters from `go-chi/httprate` wrap the route groups:
   - credential routes: 20/min keyed by IP and endpoint
   - session routes: 300/min keyed by IP
   - ingest: 600/min keyed by the SHA-256 of the bearer token
   Over the limit answers `429` with `Retry-After: 60`.
5. `/api/auth/*` is public except `me`; `/api/ingest` runs `middleware.RequireIngestAuth`;
   everything else runs `middleware.RequireAuth`, and `/api/apikeys` plus `/api/alerts` add
   `middleware.RequireAdmin`.
6. Handlers decode with `tronc/httpjson`, delegate to their service, and write typed errors
   from `tronc/errors`.

## Authentication

Two independent credentials, both `Authorization: Bearer <token>`.

**Dashboard sessions.** Issued and verified by [porte](https://github.com/FacileStudio/porte),
the suite's authentication kit, which Journal is the first app to adopt. `POST
/api/auth/register` and `/api/auth/login` return a random 32-byte URL-safe token; the single
sign-on callback puts the same thing in an HttpOnly `__Host-session` cookie instead. Only the
SHA-256 hex is stored, in `porte_sessions`, with a 30-day absolute TTL and a 7-day idle
window that applies to the cookie transport only — a bearer belongs to a CLI or a nightly
job, which is idle by design. Both transports are the same row, so one logout ends either.
Passwords are Argon2id (64 MiB, 3 iterations, parallelism 2, 16-byte salt, 32-byte key) in
the standard `$argon2id$…` encoding. A login for an unknown email runs
`authcrypto.EqualizeTiming` against a throwaway hash so response timing does not leak
account existence. Registration takes a `pg_advisory_xact_lock` and counts users inside the
transaction: the first account created becomes admin, and `ALLOW_REGISTRATION=false` only
blocks sign-ups once at least one account exists. The single sign-on callback goes through the
same lock and the same rule, in `modules/auth`'s `UserStore` — porte resolves the identity and
hands over the claims, and Journal decides what an account is.

**Ingest keys.** `POST /api/apikeys` (admin) mints `journal_<app>_<random>` and returns the
full token exactly once; the row keeps a display `prefix` and the SHA-256 hash. A key is
scoped to one `app`, so an entry whose `app` differs is rejected with `400`. Revocation sets
`revoked_at` and is idempotent, which makes rotation a matter of minting the new key,
redeploying the shipper, then revoking the old one. The legacy `INGEST_TOKEN` is compared in
constant time and is unscoped, so every entry must carry its own `app`.

## Data model

| Table | Columns of note |
|---|---|
| `porte_sessions` | `token_hash`, `user_id` → `users(id)`, `label`, `last_used_at`, `expires_at` |
| `porte_identities` | `(provider, subject)` primary key, `user_id`, the IdP tokens and the cached roles claim |
| `porte_login_codes` | one-time codes bridging a browser login to a CLI |
| `log_entries` | `app`, `level`, `message`, `meta` jsonb, `created_at`, `received_at`, generated `search` tsvector |
| `users` | `email` unique, `name`, `password_hash`, `is_admin` |
| `api_keys` | `app`, `prefix`, `key_hash` unique, `revoked_at` |
| `saved_queries` | `name` unique, `params` jsonb (`app`, `levels`, `q`, `request_id`) |
| `alert_rules` | `saved_query_id` FK, `threshold`, `window_minutes`, `webhook_url`, `webhook_header`, `webhook_secret`, `enabled`, `last_fired_at` |

`created_at` is the log's own time — the client `ts` when supplied, otherwise server receipt
time. `received_at` is always server time, so clock skew on a shipper is visible rather than
silently trusted.

`schemas.Migrate` runs GORM `AutoMigrate` and then raw SQL that GORM cannot express:

- `search` as `GENERATED ALWAYS AS (to_tsvector('simple', coalesce(message,'')))` plus a GIN
  index. The `simple` dictionary means no stemming and no stopwords, which keeps matching
  predictable across languages.
- `(app, created_at DESC)` and `(created_at DESC, id DESC)` composite indexes, the second
  backing the keyset cursor and the context query.
- A partial expression index on `(meta->>'request_id') WHERE meta ? 'request_id'`, which is
  what makes the request-id pivot cheap.
- The `alert_rules → saved_queries` foreign key with `ON DELETE RESTRICT`, so deleting a
  saved query that an alert depends on fails instead of orphaning the rule.

## Background jobs

**Retention.** When `RETENTION_DAYS > 0`, a goroutine deletes `log_entries` older than that
many days, immediately at boot and then hourly, logging the row count when it removes
anything.

**Alert evaluator.** Every minute, `alerts.RunEvaluator` loads enabled rules whose
`last_fired_at` is null or older than their window, re-runs the linked saved query over the
window, and posts a webhook when the count reaches the threshold. The payload carries the
alert and query names, the count, the threshold, the window bounds, and up to five of the
newest matching entries. `last_fired_at` is only written on a 2xx response, so a failing
webhook retries on the next tick instead of being silently swallowed.

Webhook delivery is SSRF-guarded: the default client refuses redirects and blocks dialing
loopback, link-local, multicast, unspecified, and private addresses. Hosts listed in
`WEBHOOK_ALLOWED_HOSTS` use a client without the address filter, which is how an internal
target such as Nook is reached on the compose network.

## Cross-app integration

Journal is the receiving end of the suite's logging path rather than a participant in the
`pool` / `enveloppe` event bus. Suite Go apps configure `JOURNAL_URL` and `JOURNAL_TOKEN`
through `tronc/env.Core` and tee their `slog` output with `sdk/journal`. Apps that only
write to stdout are covered by the collector sidecar instead.

`JOURNAL_URL` must end in `/api`. The SPA catch-all answers any unmatched path — a `POST`
included — with `200` and `index.html`, and both the SDK and the collector treat any 2xx as
a successful ingest. Point a shipper at the bare host and every line is accepted, discarded,
and never reported.
