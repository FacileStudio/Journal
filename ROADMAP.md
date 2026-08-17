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

## 2b. v1.3 — browser errors (the Sentry question)

The suite ships ~15 SvelteKit fronts and had **zero** visibility into what breaks in the
browser. Journal already owned ingest, search, retention and alerting, so the cheapest path to
that visibility was a second, narrower write path — not a second product.

**Shipped.**

- [x] **Public ingest keys** — `api_keys.kind` splits a server credential from a key meant to
      be pasted into a bundle. A public key carries an exact origin allowlist (1–8 entries, no
      wildcards, normalized to what a browser actually sends) and a mandatory daily quota, and
      each kind authenticates exactly one route. Minted from Settings → API.
- [x] **`POST /ingest/browser`** — 20 events, 128 KB, no `app` in the payload (the key's app is
      authoritative), `meta` scrubbed of credential-shaped keys, depth-limited and capped at
      8 KB, with `source`/`origin`/`user_agent` stamped server-side. The key rides in `?key=`
      so `sendBeacon` works, and the body is `text/plain` so no preflight is involved.
- [x] **Daily quota** — reserved before the write in one conditional upsert, so two concurrent
      requests cannot both read "just under the limit". 429 with `Retry-After` to UTC midnight.
- [x] **`@facile/journal`** (`sdk/browser`) — dependency-free browser SDK: window handlers, a
      SvelteKit `handleError`, 60s dedup with counts, noise filter, sampling, per-session cap,
      beacon on page hide, mute on 429, retry on network failure.

**Not built, and deliberately so.** These are what separate a log tool from Sentry, and each
one only pays at a volume this instance does not have yet:

| Feature | Trigger |
|---|---|
| **Issue grouping** — `issues` table, fingerprint = type + normalized message + top in-app frame, `first_seen`/`last_seen`/status, a list-of-issues UI | the error stream is too noisy to read as a stream. Watch for a single fingerprint dominating a day |
| ~~**Source maps**~~ | **Shipped 2026-08-11.** Uploaded by the app at boot from inside its own image, keyed on the release in `_app/version.json`; resolved on read via `GET /logs/{id}/stack`. Sablier is the first consumer |
| **Breadcrumbs** — clicks, navigations, failed fetches, console, in a 50-entry ring | triage keeps ending in "I cannot reproduce it" |
| **Wildcard origins** (`https://*.facile.studio`) | preview deployments per branch make an 8-entry exact list impractical |
| **Preflight support** on `/ingest/browser` | a consumer that cannot send `text/plain`. Needs the app-wide CORS middleware to defer to the key's allowlist, which today it does not |

If two of those trigger at once, reconsider **GlitchTip** (self-hosted, Sentry-protocol,
OIDC against porte) before building the third — the argument for building was that the first
90% was nearly free, and it stops applying once grouping and source maps are on the table.

### Closed: the per-IP rate limits were spoofable — fixed in tronc v0.10.0

- [x] `tronc/httpx` applied chi's `middleware.RealIP`, which rewrites `RemoteAddr` from
      `X-Forwarded-For` **without checking who the peer is**, so every per-IP limit in this app
      — login, session routes, browser ingest — was bypassable by rotating a header. Measured
      against a local build: 70 requests with a rotating `X-Forwarded-For` all returned 201
      where the 60/min bucket should have refused 10.

      Fixed in **tronc v0.10.0** with `middleware.RealIP(trusted)`, which believes the header
      only from a trusted peer and walks the chain right to left. Journal is on it, and
      `internal/middleware/realip.go` — the local implementation nothing ever called — is
      deleted. Re-measured after the bump with the app's trust set pointed away from the
      caller: the same 70 requests give **60 accepted, 10 refused**. `TRUSTED_PROXIES` narrows
      the set; unset is loopback plus the private ranges, so prod needed no new variable.

      The browser endpoint keeps its per-key ceiling and daily quota anyway. A bound that owes
      nothing to the network layer is worth having even when the network layer is correct.

## 2c. v1.4 — context around a browser error

A stack trace answers *what* threw and never *what led to it*, which is where triage kept
stalling. Two layers shipped ahead of breadcrumbs because both are useful on their own, both are
an order of magnitude smaller, and having them makes the breadcrumb question easier to judge
honestly rather than by enthusiasm.

- [x] **Session id** — every browser batch carries one id per tab, kept in `sessionStorage` so it
      survives a reload (a reload is regularly the bug), stored as `meta.session_id` and
      filterable through `GET /logs?session_id=` with a pivot beside `request_id`. It rides on the
      batch, not the event, and it is a wire field rather than a meta key for two reasons:
      `session` is on the scrub list, and an event that could name its own session could file
      errors under another tab's. Deliberately **not** available to saved queries or alert rules —
      a rule pinned to a dead tab is noise.
- [x] **Fetch tracing** (`trace` option, off by default) — wraps `fetch`, sends `X-Request-Id`,
      reports 5xx and network failures as `kind: 'fetch'` events carrying `meta.request_id`, which
      is the key the explorer already pivots on. 4xx is not reported: an expired session and a 404
      probe are the application working. Same-origin unless an origin is named, because a custom
      header makes a request non-*simple* and earns a preflight the other server has to answer.
      Journal's own API is never traced.

Both landed with the same constraint in view, and it governs everything below: **`browserMeta`
caps the whole meta object at 8 KB** and falls back to a five-key map, silently. Arrays are cut at
50 and strings at 2 KB. Nothing large can ride in `meta`; it needs its own wire field, its own
budget, and an explicit place in the fallback.

### The server half: shipped in tronc v0.14.0

- [x] **`middleware.RequestID`** — accepts a well-formed inbound `X-Request-Id`, mints an opaque
      one otherwise, **echoes it on the response**, and keeps chi's context key so `GetReqID`,
      `RequestLogger` and `Recoverer` need no changes anywhere. `CORS` gained `ExposedHeaders`
      (default `X-Request-Id`) and allows the header inbound. Journal is on it as of the bump to
      v0.14.0.

It went in tronc rather than here for the same reason `RealIP` did: one version bump lights it up
in every app on the chassis. Two defects in chi's middleware came out while writing it, and both
had been live in every one of those apps — the header was taken **verbatim** into every log line
(unbounded, any bytes, and in this app it lands in `meta.request_id` and a clickable filter), and
chi's minted id embeds `os.Hostname()`, which is harmless until something echoes it. See tronc's
CHANGELOG 0.14.0.

**Still one-way until an app bumps.** Journal itself is on v0.14.0; every other suite app is on
0.12/0.13, so a browser error from *those* fronts still carries only the id its own SDK minted.
The remaining work is a `go get github.com/FacileStudio/tronc@v0.14.0` per app, not more design.

### Still not built here

| Feature | Trigger |
|---|---|
| **Device envelope** — viewport, DPR, language, referrer, connection type in `meta` | the next time "it only breaks on narrow screens" costs a round trip to ask. Small enough to ride in `meta` as-is |
| **Breadcrumbs** (see §2b) | unchanged — but re-read that table's GlitchTip rule first. Breadcrumbs need a first-class wire field, their own budget, a raised body cap and a drawer timeline; they are not a `meta` key |

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
