# Journal Roadmap

Informed by a full implementation review (2026-07-05) and a survey of small-scale log tools
(Seq, Loki, VictoriaLogs, Dozzle, Papertrail, OpenObserve) plus Postgres-as-log-store and
ingest API best practices. Ordered by value per line of code. YAGNI applies: the "Later"
drawer stays shut until its trigger fires.

## 0. Fixes before features (correctness & security debt)

Found in review; none require design work.

### High

- [x] **Cursor pagination drops rows** — `GET /logs` sorts by `(created_at desc, id desc)` but
      the cursor is `id < before`. A late-arriving entry with an old client `ts` and a high id
      is skipped forever. Fix: row-value keyset cursor `(created_at, id) < (?, ?)`; the cursor
      carries both values. Update CLAUDE.md contract (it documents the broken scheme).
      `apps/api/modules/logs/service.go`, `handler.go`.
- [x] **`docker-compose.override.yml` is auto-loaded everywhere** — it publishes Postgres on
      `0.0.0.0:5432` (hardcoded fallback password) and the API on `:4010` on any host that runs
      `docker compose up` from a checkout, including a VPS. Rename to `docker-compose.dev.yml`
      and use `-f` locally, or gate with a profile.
- [x] **Un-pinned CDN script + no CSP** — `app.html` loads `iconify-icon` from jsdelivr with no
      SRI, and the SvelteKit pages ship zero security headers (API middleware only covers
      `/api/*`). This is the sink that turns the localStorage-token tradeoff into a real
      exploit. Vendor the file into `static/` (or add `integrity` + `crossorigin`), and add a
      `hooks.server.ts` handle that sets CSP, `X-Frame-Options`, `X-Content-Type-Options`.

### Medium

- [x] **`INGEST_TOKEN: ${INGEST_TOKEN:-change-me}`** in docker-compose.yml defeats the
      documented fail-closed behavior (empty → reject all). Drop the fallback.
- [x] **No `restart:` policy** on any service — stack stays down after reboot/OOM. Add
      `restart: unless-stopped`.
- [x] **Fatal startup errors exit 0** — `main.go` logs and `return`s on config/DB/migration/
      listen failure; restart policies and Dokploy see success. Use `os.Exit(1)`.
- [x] **>1MB message wedges a whole batch** — no cap on `message`, but the generated tsvector
      column has a 1MB Postgres hard limit; one pathological line 500s and rolls back up to
      500 entries. Cap message at 64KB (truncate) in ingest validation.
- [x] **First-user-admin race + registration bootstrap TOCTOU** — count check runs at READ
      COMMITTED; two concurrent registers both become admin. Take a transaction-level advisory
      lock (`pg_advisory_xact_lock`) around first-user logic.
- [x] **Expired sessions never deleted** — expiry is enforced on read but rows accumulate
      forever. Delete expired rows opportunistically on login.
- [x] **Rate limiter trusts spoofable `X-Forwarded-For`** (chi `RealIP` before
      `httprate.LimitByIP`) and also throttles `/health`, `/ready`, and `/ingest` (a chatty
      non-batching app silently loses logs at 100/min). Trust XFF only from Traefik, exempt
      health checks, give ingest its own (higher) bucket.
- [x] **Dashboard load/poll races** — no AbortController on debounced loads (stale response
      overwrites newer filter) and live-tail poll can prepend old-filter rows into a new
      filter view. Guard with a request generation counter or AbortController.
      `(app)/+page.svelte`.
- [x] **Live tail grows `entries` unboundedly** — cap the array (e.g. 2000 rows) when
      prepending fresh entries.

### Low / cleanup

- [x] Proxy: delete `content-encoding`/`content-length` response headers too (undici already
      decompresses — latent bug if the API ever adds gzip); add try/catch + timeout on
      upstream fetch; re-encode path segments.
- [x] Duplicate-email race returns 500 → map `gorm.ErrDuplicatedKey` to 409.
- [x] `{"entries": []}` returns a misleading 400 → return `{"ingested": 0}`.
- [x] Set DB pool limits (`SetMaxOpenConns` ~10, `SetConnMaxLifetime`).
- [x] Clamp client `ts` to `received_at ± 24h` (a `ts` in year 9999 pins the top of every list).
- [x] `bun install --frozen-lockfile` in client Dockerfile; add `.dockerignore`.
- [x] Resolve the CLAUDE.md contradiction: in prod, Traefik routes `/api` straight to the Go
      API — the SvelteKit proxy is dev-only plumbing, not "the only public surface". Pick one
      story and document it.

## 1. v1.1 — the features that make it a real log tool

- [x] **Retention job** — nightly `DELETE FROM log_entries WHERE created_at < now() - interval
      '90 days'` (config `RETENTION_DAYS`, 0 = keep forever). A goroutine ticker in the API is
      enough; no partitioning at this volume. Universal in every surveyed tool; the table
      currently grows forever.
- [x] **Per-app API keys** — replace the single `INGEST_TOKEN` with an `api_keys` table
      (`app`, `key_hash` SHA-256, `created_at`, `revoked_at`), token format
      `journal_<app>_<random>`, multiple active keys per app so rotation is add-new → redeploy
      → revoke-old with zero downtime. Admin-only CRUD endpoints + a small dashboard page.
      Gives per-app attribution and kill-switches for free. (Seq's model.)
- [x] **Time-range filter UI** — `since`/`until` already exist in the API and in `backend.ts`;
      the dashboard just never exposes them. Cheapest feature on this list.
- [x] **Level histogram** — count-by-level-over-time bar chart above the log list, scoped to
      the current filter (`date_trunc` + `GROUP BY` endpoint). The single highest-value
      visualization in Seq/OpenObserve/Grafana: "something started erroring at 14:32".
- [x] **`request_id` correlation** — promote `meta->>'request_id'` to a clickable pivot: click
      an id, see that request's logs across all apps. Expression index
      `((meta->>'request_id'))` (btree, not jsonb GIN — GIN has real write amplification and
      containment-only payoff). This is the feature centralized logging exists for.
- [x] **Context view** — from a search hit, jump to the surrounding unfiltered stream
      (`created_at BETWEEN match ± interval`, or ±N ids). Papertrail's best trick; one query.

## 2. v1.2 — comfort and robustness

- [x] **Saved queries** — tiny `saved_queries` table (name, filter params), a dropdown in the
      dashboard. Prerequisite for alerting.
- [x] **Webhook alerts** — evaluate saved queries every N minutes; if count > threshold, POST
      to a webhook (Nook). Skip Alertmanager-style routing; one URL per rule.
- [x] **Ingest hardening** — accept `Content-Encoding: gzip` (stdlib `gzip.NewReader`), cap
      batches at 1000 entries → 400, return 429 + `Retry-After` under pressure. Document
      retryable statuses (429/5xx) in the shipper snippet.
- [x] **Tail ergonomics over transport** — pause button, filter-while-tailing, "may have
      missed logs" marker when a poll returns a full page (100), clickable fields (app, level,
      request_id) that pivot the filter. Polling at 2.5s is fine for one user — VictoriaLogs
      polls its own storage at 1s internally. If upgrading anyway: SSE (`http.Flusher` +
      `EventSource`), never WebSocket.
- [x] **Docker log collector sidecar** — tail container json-file logs on la ruche and ship to
      `/ingest`, so apps that only `console.log` are captured with zero code change.

## 3. Later drawer (open only on trigger)

| Feature | Trigger |
|---|---|
| OTLP `/v1/logs` endpoint (`go.opentelemetry.io/proto/otlp`, leaf dep, one handler) | first OTel-instrumented app needs it |
| Monthly partitioning + drop-partition retention (pg_partman or native) | `log_entries` ≥ ~10 GB |
| BRIN on `created_at` | pure time-range scans get slow on a huge table |
| VictoriaLogs / ClickHouse migration (keep HTTP contract) | ~100M rows or aggregations time out |
| Cookie-based dashboard sessions (`HttpOnly; Secure; SameSite`) | suite-wide auth decision — OWASP recommends it over localStorage, but changing it means diverging from the shared Nuage pattern; decide once for the suite, not per app |

## Explicit non-goals

Multi-tenancy, RBAC, clustering, log parsing/pipelines, dedup on ingest (shippers are
at-least-once; dupes are acceptable at this scale), Vault/KMS (env vars + Postgres are
correct at this size).
