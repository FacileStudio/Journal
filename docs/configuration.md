# Journal — Configuration

Every environment variable the code actually reads, grouped by the process that reads it.

## API

`apps/api/internal/env` builds its config on top of `tronc/env.LoadCore`, so Journal
inherits the suite's core variables and adds four of its own. A missing or invalid required
variable makes the process log the failure and exit 1 before the server ever binds.

### Core, from `tronc/env`

| Variable | Required | Default | What it does |
|---|---|---|---|
| `DATABASE_URL` | yes | — | Postgres connection string. Absent or blank exits 1 |
| `PORT` | no | `8080` | HTTP listen port. Rejected unless it parses to 1–65535. The shipped compose file sets `4010` |
| `LOG_LEVEL` | no | `info` | `debug`, `info`, `warn`, or `error`. Anything else means `info` |
| `APP_ENV` | no | `development` | `development`, `staging`, or `production`. Never gates security behavior |
| `CORS_ALLOWED_ORIGINS` | no | — | Comma-separated origins. Unset denies every cross-origin caller, which is correct for the single-container deployment |
| `JOURNAL_URL` | no | — | Part of `tronc/env.Core`; Journal does not ship logs to itself, so it is parsed and unused here |
| `JOURNAL_TOKEN` | no | — | Same as above |

`tronc` reads the CORS origin list from the first name that is set, in this order:
`CORS_ALLOWED_ORIGINS`, `ALLOWED_ORIGINS`, `DOMAINS`, `DOMAIN`, `CORS_ORIGINS`,
`TRUSTED_ORIGINS`, `CLIENT_ORIGIN`. The first is canonical; the rest exist so a repo can
adopt `tronc` without touching its deployment config. The shipped `docker-compose.yml`
passes `ALLOWED_ORIGINS`.

### Journal's own

| Variable | Required | Default | What it does |
|---|---|---|---|
| `INGEST_TOKEN` | no | — | Legacy unscoped bearer token for `POST /api/ingest`. Empty disables it, leaving per-app API keys as the only ingest credential |
| `ALLOW_REGISTRATION` | no | `true` | `false` blocks dashboard sign-ups once at least one account exists. The first account is always allowed, so you cannot lock yourself out. Must parse as a boolean |
| `RETENTION_DAYS` | no | `90` | Delete entries older than N days, hourly. `0` keeps forever. A negative value exits 1 |
| `WEBHOOK_ALLOWED_HOSTS` | no | — | Comma-separated hostnames alert webhooks may reach past the SSRF guard. Without it, private, loopback, link-local, and metadata addresses are refused |

### Single sign-on

Journal federates to the suite's Authentik through [porte](https://github.com/FacileStudio/porte).
The variable names are the suite convention and are the same in every Facile app.

| Variable | Required | Default | What it does |
|---|---|---|---|
| `OIDC_ISSUER` | no | — | Issuer URL. Setting it enables SSO and turns the next four into hard requirements. Must be an absolute `http(s)` URL — a bare hostname parses as a relative path and would otherwise fail discovery at boot with an opaque error |
| `OIDC_CLIENT_ID` | with issuer | — | Client identifier |
| `OIDC_CLIENT_SECRET` | with issuer | — | Client secret |
| `OIDC_REDIRECT_URL` | with issuer | — | Callback URL, e.g. `https://journal.facile.studio/api/auth/oidc/callback`. Must match the provider's registration exactly |
| `OIDC_SUCCESS_URL` | with issuer | — | Where the browser lands after a successful login |
| `SSO_ONLY` | no | `false` | `true` stops registering `/auth/register` and `/auth/login`. Requires `OIDC_ISSUER`: without one it would leave no way to sign in, so the API exits 1 rather than lock you out |

`OIDC_ISSUER` is all-or-nothing, and discovery runs at startup: an unreachable issuer or a
missing secret exits 1 there rather than becoming a 500 on somebody's first login three days
later. Leaving it unset registers no OIDC endpoint at all.

### Static client

| Variable | Required | Default | What it does |
|---|---|---|---|
| `CLIENT_DIR` | no | `./client` | Directory holding the built dashboard, read by `tronc/spa.DirFromEnv`. The SPA route is skipped entirely when the directory has no `index.html` |

The image sets `CLIENT_DIR=/client` explicitly. The distroless `:nonroot` base carries its
own working directory, so a relative `./client` would resolve somewhere else and the
dashboard would silently not be served.

`PORT` is read a second time by `tronc/healthcheck`, which is what the compose healthcheck
`["CMD", "/api", "healthcheck"]` invokes.

## Collector sidecar

`apps/collector` is a separate module with a separate, smaller set.

| Variable | Required | Default | What it does |
|---|---|---|---|
| `JOURNAL_URL` | no | `http://journal-api:4010/api` | Journal API base URL, `/api` included |
| `JOURNAL_TOKEN` | yes | — | Ingest token. Empty logs `JOURNAL_TOKEN is required` and exits 1 |
| `DOCKER_SOCK` | no | `/var/run/docker.sock` | Docker Engine socket to read container logs from |
| `DISCOVER_INTERVAL` | no | `30` | Seconds between container discovery sweeps. Ignored unless it parses to a positive integer |

The collector ships lines for many containers, and a per-app API key is scoped to a single
`app` name — it would reject every other container. `JOURNAL_TOKEN` must therefore be the
legacy unscoped `INGEST_TOKEN`, which means enabling the collector requires setting
`INGEST_TOKEN` even though per-app keys are otherwise preferred.

## Compose substitutions

These never reach the Go processes; `docker-compose.yml` interpolates them.

| Variable | Default | What it does |
|---|---|---|
| `POSTGRES_USER` | `journal` | Internal Postgres user, also spliced into `DATABASE_URL` |
| `POSTGRES_PASSWORD` | `journal-internal-db` | Internal Postgres password |
| `POSTGRES_DB` | `journal` | Internal database name |
| `COMPOSE_PROFILES` | — | Set to `collector` to also start the collector sidecar |

## Traps

- **A missing `DATABASE_URL` exits 1.** So does a non-numeric `PORT`, a non-boolean
  `ALLOW_REGISTRATION`, and a negative `RETENTION_DAYS`. The failure is logged before the
  listener starts, so the container restart loop is the first symptom.
- **`JOURNAL_URL` must end in `/api`** for any shipper, the collector and the Go SDK
  included. The SPA catch-all answers unmatched paths with `200` and `index.html`, so a base
  URL missing the prefix looks like a successful ingest while every line is dropped.
- **No ingest credential means no ingest.** With `INGEST_TOKEN` empty and no API keys
  created, every `POST /api/ingest` is a `401`.
- **`.env.example` is a compose-level template**, not the API's own. The API's variables
  have their own template in `apps/api/.env.example`. Neither file is the source of truth —
  this page is.
