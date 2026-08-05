# Journal — Development

Local setup, the test layout, and the quality gate that runs before every push.

## Prerequisites

| Tool | Version | Why |
|---|---|---|
| Go | 1.24 | Pinned by `mise.toml` and by `go 1.24.0` in all three `go.mod` files |
| Bun | 1.x | Client install, dev server, type-check, and build |
| Docker | any recent | Postgres, and the full-stack compose run |
| mise | optional | Task runner; every task is a thin wrapper over a script you can run directly |

## Setup

Enable the tracked git hooks once per clone:

```sh
mise run hooks
```

That is `git config core.hooksPath .githooks`. Without it the pre-push gate never runs.

Install the client dependencies:

```sh
mise run install
```

## Running

Start Postgres alone, then each half in its own terminal:

```sh
docker compose -f docker-compose.yml -f docker-compose.dev.yml up journal-db -d
```

```sh
cd apps/api
cp .env.example .env
go run .
```

```sh
cd apps/client
bun run dev
```

The API listens on `4010`, chosen so it does not collide with Nuage on `4000`. The client
dev server is on `5173` and Vite proxies `/api` to `http://localhost:4010`, so the browser
sees the same single-origin shape as production.

To run the whole stack in containers instead:

```sh
docker compose -f docker-compose.yml -f docker-compose.dev.yml up --build
```

The dev overlay is what publishes host ports — `127.0.0.1:4010` for the app and
`127.0.0.1:5432` for Postgres. Plain `docker compose up` publishes nothing, which is the
production shape.

## Repository layout

Three independent Go modules, each with its own `go.mod`:

| Module | Path |
|---|---|
| `github.com/FacileStudio/Journal/apps/api` | the server |
| `github.com/FacileStudio/Journal/apps/collector` | the Docker log sidecar, stdlib only |
| `github.com/FacileStudio/Journal/sdk/journal` | the client SDK, stdlib only, `go get`-able |

Splitting them keeps the SDK's dependency graph empty: an app that imports it does not
inherit Chi, GORM, or the Postgres driver.

API modules follow the same four-file shape as the rest of the Go family —
`modules/<name>/router.go` exposing `RegisterRoutes`, plus `handler.go`, `service.go`, and
`types.go`. GORM models live in `apps/api/schemas/`, and `schemas/migrate.go` owns both the
`AutoMigrate` call and the raw SQL that follows it.

The client is Svelte 5 runes only — `$state`, `$props`, `$derived`, `$effect` — with
TypeScript in strict mode, and every API call goes through `src/lib/backend.ts`.

## Tests

```sh
cd apps/api && go test ./...
```

Test files sit beside the code they cover: `authcrypto/crypto_test.go` for password and
token handling, `modules/ingest/handler_test.go` for entry validation and batching,
`modules/logs/handler_test.go` for filter and cursor parsing, `modules/apikeys`,
`modules/queries`, and `modules/alerts` for their services, plus
`modules/alerts/evaluator_test.go` and `safeguard_test.go` for the evaluator and the SSRF
guard. The collector has `entry_test.go` and `stream_test.go` for Docker stream framing and
level detection; the SDK has `journal_test.go`.

Client type-check:

```sh
cd apps/client && bun run check
```

## Quality gate

```sh
mise run check        # every Go module, then the client
mise run check-go     # Go only
mise run format       # rewrite Go sources in place
```

All three call `scripts/check.sh`, which runs `gofmt -l`, `go vet ./...`, and `go test ./...`
in `apps/api`, `apps/collector`, and `sdk/journal`, then `bun run check` in `apps/client`.
It reports and never rewrites, except under `--format`.

`.githooks/pre-push` execs the same script, so a push runs the full gate. Bypass a
known-unrelated failure with `git push --no-verify`.

Two deliberate details in the script are worth knowing before you edit it. It is invoked as
`sh scripts/check.sh` rather than through `mise run`, because `mise run` resolves every tool
in the merged config first and an unrelated broken tool in a global config would take the
gate down with it. And it prefers `$GOROOT/bin/go` when `GOROOT` is set, because mise
exports `GOROOT` for the pinned version while leaving another `go` earlier on `PATH`, which
fails with `compile: version "X" does not match go tool version "Y"`.
