# Journal Collector

Docker log collector sidecar. Tails stdout/stderr of every running container on the host via
the Docker Engine API (`/var/run/docker.sock`) and ships the lines to Journal's `POST /ingest`,
so apps that only `console.log` get centralized logs with zero code change.

Go stdlib only — no Docker SDK. Handles both multiplexed (non-TTY) and raw (TTY) log streams,
parses per-line Docker timestamps, detects levels from JSON log lines (`level`/`lvl`/`severity`),
and falls back to stdout → `info`, stderr → `error`. Entries carry
`meta.container_id` (12 chars) and `meta.stream`.

## Token: legacy `INGEST_TOKEN` required

The collector ships logs for **many** apps, but per-app API keys are scoped to a single app —
a scoped key would reject every other container with a 400. `JOURNAL_TOKEN` must therefore be
Journal's legacy **unscoped** `INGEST_TOKEN`. Set `INGEST_TOKEN` in the deploy env (empty
disables legacy ingest entirely, and the collector refuses to start without a token).

## Enabling

The compose service is behind the `collector` profile — plain `docker compose up` does not
start it:

```sh
COMPOSE_PROFILES=collector docker compose up -d --build
```

In production, set `COMPOSE_PROFILES=collector` in the deploy environment.

## Container labels

| Label | Effect |
|---|---|
| `journal.ignore=true` | Container is never tailed (the collector labels itself with this) |
| `journal.app=<name>` | Overrides the `app` field (default: container name) |

## Configuration

| Variable | Description | Default |
|---|---|---|
| `JOURNAL_URL` | Journal API base URL | `http://journal-api:4010` |
| `JOURNAL_TOKEN` | Legacy unscoped ingest token (required) | — |
| `DOCKER_SOCK` | Docker Engine socket path | `/var/run/docker.sock` |
| `DISCOVER_INTERVAL` | Container discovery interval, seconds | `30` |

## Caveats

- **Restart loss**: on collector restart, tailing starts at "now" — lines logged while the
  collector was down are not backfilled. Within a running session, tails reconnect from the
  last seen timestamp. Batches are buffered in memory (max 5000 entries, oldest dropped) and
  retried on 429/5xx/network errors; a 400 drops the batch so a poison batch can't wedge the
  pipeline.
- **Runs as root**: reading `/var/run/docker.sock` requires root or the host's `docker` group;
  the group's GID isn't knowable at image build time, so the container runs as root. The socket
  is mounted read-only and the collector only issues `GET` requests against it.
- New containers are picked up on the next discovery sweep (up to `DISCOVER_INTERVAL` seconds),
  and tailing starts from that moment — a container's very first lines can be missed.
