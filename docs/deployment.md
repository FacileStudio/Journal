# Journal — Deployment

How the image is built, what Compose starts, and how Traefik routes to it on la ruche.

## Image

A single root `Dockerfile` with the repo root as its build context, in three stages:

1. `oven/bun:1` installs `apps/client` with `--frozen-lockfile` and runs `bun run build`.
2. `golang:1.24-alpine` downloads the API module's dependencies and builds a static binary
   with `CGO_ENABLED=0` and `-trimpath -ldflags="-s -w"`.
3. `gcr.io/distroless/static-debian12:nonroot` receives the binary at `/api` and the built
   client at `/client`, sets `CLIENT_DIR=/client`, exposes `4010`, and runs as
   `nonroot:nonroot`.

`CLIENT_DIR` is set explicitly because the distroless `:nonroot` variant carries its own
working directory; a relative `./client` would resolve there and the dashboard would
silently not be served. The `:nonroot` variant is safe only because the app service mounts
no writable volume — adding one means dropping back to the root distroless variant.

## Compose topology

```
dokploy-network ──▶ journal-api  (:4010, Traefik-routed, no host ports)
                          │
default network ──────────┴──▶ journal-db (postgres:16-alpine, expose only)
                          └──▶ journal-collector (profile: collector, optional)
```

| Service | Notes |
|---|---|
| `journal-db` | `postgres:16-alpine`, named volume `journal_db_data`, `pg_isready` healthcheck every 5s |
| `journal-api` | Built from the root `Dockerfile`, joins both `default` and the external `dokploy-network`, waits on `journal-db` being healthy |
| `journal-collector` | Behind the `collector` profile, mounts `/var/run/docker.sock` read-only, waits on `journal-api` being healthy |

Postgres publishes no ports at all — it only `expose`s `5432` on the compose network, and
`DATABASE_URL` is assembled inside the compose file from `POSTGRES_USER`,
`POSTGRES_PASSWORD`, and `POSTGRES_DB`. `docker-compose.dev.yml` is the only thing that
binds host ports, on `127.0.0.1`.

The API healthcheck is `["CMD", "/api", "healthcheck"]` — the binary's own subcommand,
resolved through `tronc/healthcheck`, which reads `PORT`. It must track the binary's path
inside the image if that ever moves.

## Traefik

One hostname, one service, two routers (plain HTTP redirecting to TLS):

```yaml
traefik.enable: "true"
traefik.docker.network: dokploy-network
traefik.http.routers.journal-web.rule: "Host(`journal.facile.studio`)"
traefik.http.routers.journal-web.entrypoints: web
traefik.http.routers.journal-web.middlewares: redirect-to-https@file
traefik.http.routers.journal-web.service: journal-svc
traefik.http.routers.journal-secure.rule: "Host(`journal.facile.studio`)"
traefik.http.routers.journal-secure.entrypoints: websecure
traefik.http.routers.journal-secure.tls.certresolver: letsencrypt
traefik.http.routers.journal-secure.service: journal-svc
traefik.http.services.journal-svc.loadbalancer.server.port: "4010"
```

This is the suite's one-container, one-router, one-hostname rule. There is no `PathPrefix`
and no strip-prefix middleware: the `/api` prefix is part of the Go router itself, so
`/api/ingest` arrives as `/api/ingest`. That matters because the earlier two-container split
had Traefik strip `/api` before forwarding; keeping the public URLs identical is what let
the suite apps and the collector keep pointing at
`JOURNAL_URL=https://journal.facile.studio/api` unchanged.

## Deploying on la ruche

Deployments are managed through Dokploy at `gare.facile.studio`, which owns the environment
file and triggers the Compose build. Prefer the `dokploy` CLI over SSH plus `docker`.

Set at minimum `POSTGRES_USER` and `POSTGRES_PASSWORD`. Set `APP_ENV=production`
and `RETENTION_DAYS` to whatever the
disk can carry. Leave `INGEST_TOKEN` empty unless the collector is enabled.

To run the collector, set both `COMPOSE_PROFILES=collector` and `INGEST_TOKEN` — the sidecar
needs the legacy unscoped token and refuses to start without one.

## Migrations

The schema is ordered goose SQL in `apps/api/migrations/`, embedded in the binary and applied
by `tronc/migrate` at boot, before the listener binds, behind a Postgres advisory lock. A
failed migration exits 1 rather than serving against a half-migrated schema. There is no
separate migration step.

`00001_baseline.sql` is the schema v0.4.x built with GORM `AutoMigrate` and idempotent SQL,
as `pg_dump --schema-only` printed it from production on 2026-11-13. A database those
versions built (Journal's tables, no applied migration) is stamped at version 1 on its first
boot, so the baseline is recorded without running. A database older than v0.4.0 is refused:
boot v0.4.3 once, then this release.

The binary answers `migrate status`, `migrate version`, `migrate up` and `migrate baseline <n>`
without a shell:

```sh
docker exec journal-api /api migrate status
```

Rolling back to v0.4.x runs its `AutoMigrate`, which drops the timestamp defaults
`00002_timestamp_defaults.sql` added while the ledger still records version 2. Before
deploying v0.5.0 or later again after such a rollback, run
`DELETE FROM goose_db_version WHERE version_id = 2`.

## Health and readiness

`/health` and `/ready` answer at both the root and under `/api`, so the same probe works
whether the edge forwards everything or only `/api/*`. `/health` returns
`{"status":"ok"}` as soon as the process is serving and touches no dependency; `/ready`
pings Postgres with a 2-second timeout and returns `503 {"status":"not_ready"}` when it
fails. Use `/health` for restarts and `/ready` for traffic.

A green `/api/health` says nothing about the dashboard — it only proves the Go process is
up. Verify a deploy by loading the site root and confirming the SPA shell renders, not just
by curling the health endpoint.
