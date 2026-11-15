# Journal Collector

Docker log collector sidecar. Tails stdout/stderr of every running container on the host via
the Docker Engine API (`/var/run/docker.sock`) and ships the lines to Journal's `POST /ingest`,
so apps that only `console.log` get centralized logs with zero code change.

Go stdlib only — no Docker SDK. Handles both multiplexed (non-TTY) and raw (TTY) log streams,
parses per-line Docker timestamps and detects levels from JSON log lines
(`level`/`lvl`/`severity`), logfmt (`level=`), a leading Postgres token (`LOG:`, `NOTICE:`,
`WARNING:`, `ERROR:`, `FATAL:`, `PANIC:`, optionally after a timestamp and `[pid]`) and a leading
bracketed level (`[ERROR]`). A line with no recognizable level becomes `info` on stdout and
`warn` on stderr, because many programs write ordinary output to stderr. Entries carry
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
| `journal.app=<name>` | Overrides the `app` field. The default is the container name, without the replica number Compose appends (`facile-sablier-lbgdwy-api-19` ships as `facile-sablier-lbgdwy-api`), so an app keeps one name across rolling deploys. An explicit `container_name` is kept as is. |

## Configuration

| Variable | Description | Default |
|---|---|---|
| `JOURNAL_URL` | Journal API base URL, `/api` included | `http://journal-api:4010/api` |
| `JOURNAL_TOKEN` | Legacy unscoped ingest token (required) | — |
| `DOCKER_SOCK` | Docker Engine socket path | `/var/run/docker.sock` |
| `DISCOVER_INTERVAL` | Container discovery interval, seconds | `30` |
| `COLLECT_MIN_LEVEL` | Per-app minimum level, `app=level,app=level`. Lines below it are dropped before shipping. Levels: `debug`, `info`, `warn`, `error`. An `app` key matches the exact app name, or any app name that starts with `<key>-` (so `labparasite-apppupille` covers `labparasite-apppupille-6bypjx-api-1`). Invalid pairs are ignored. | unset (ship everything) |
| `COLLECT_SDK_APPS` | Apps that already ship their logs with the Journal SDK, comma-separated, matched like `COLLECT_MIN_LEVEL` keys. For these apps the collector drops slog JSON lines (an object with `msg` and `level`), because the SDK sent the same record with its structured fields. Panics, plain-text lines and other JSON stay collected, so crash output is never lost. | unset (collect everything) |

## Caveats

- **`JOURNAL_URL` must include `/api`**: the API serves its routes under `/api` and the SPA
  catch-all answers everything else with `200` + `index.html`. A base URL missing the prefix
  therefore looks like a successful ingest to the shipper, and every line is dropped in
  silence.
- **Blank lines are not shipped**: the API rejects an empty `message` with `400` and drops the whole batch, so the collector skips whitespace-only lines.
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
