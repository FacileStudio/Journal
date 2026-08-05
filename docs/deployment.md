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

Set at minimum `POSTGRES_USER` and `POSTGRES_PASSWORD`. Set `APP_ENV=production`,
`ALLOW_REGISTRATION=false` once your accounts exist, and `RETENTION_DAYS` to whatever the
disk can carry. Leave `INGEST_TOKEN` empty unless the collector is enabled.

To run the collector, set both `COMPOSE_PROFILES=collector` and `INGEST_TOKEN` — the sidecar
needs the legacy unscoped token and refuses to start without one.

## Migrations

There is no migration tool and no separate migration step. `schemas.Migrate` runs at boot,
before the listener binds: GORM `AutoMigrate` over the six models, then idempotent raw SQL
for the generated `search` column, the GIN and composite indexes, the partial `request_id`
index, and the `alert_rules` foreign key. Every statement is `IF NOT EXISTS` or a
drop-then-add, so a redeploy is safe and a failure exits 1 rather than serving against a
half-migrated schema.

## Health and readiness

`/health` and `/ready` answer at both the root and under `/api`, so the same probe works
whether the edge forwards everything or only `/api/*`. `/health` returns
`{"status":"ok"}` as soon as the process is serving and touches no dependency; `/ready`
pings Postgres with a 2-second timeout and returns `503 {"status":"not_ready"}` when it
fails. Use `/health` for restarts and `/ready` for traffic.

A green `/api/health` says nothing about the dashboard — it only proves the Go process is
up. Verify a deploy by loading the site root and confirming the SPA shell renders, not just
by curling the health endpoint.
