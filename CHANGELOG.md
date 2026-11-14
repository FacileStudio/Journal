# Changelog

## [Unreleased]

### Added

- feat(api): `RETENTION_MAX_ENTRIES` (default 10,000,000, `0` turns it off) caps `log_entries`: above it the hourly retention job deletes the oldest received entries first, so one chatty producer cannot fill the disk inside the 90-day window. Production took in about 800,000 entries a day on 2026-11-14, which a 90-day window alone would let grow to about 80 GB. The bound keeps `id > max(id) - N`, one primary-key lookup per hour. The compose file forwards the variable.
- feat(api): resolved or expired MCP confirmations (`mcp_actions`) are deleted 30 days later, hourly, through tronc's `mcp.KeepPruned`.

### Changed

- refactor(api): the retention job moves from `main.go` to `internal/retention` and deletes in batches of 10,000 rows, so a backlog after lowering `RETENTION_DAYS` is never one long transaction, and a shutdown stops it between batches. The boot log also warns when `RETENTION_DAYS` is above 365 (CNIL deliberation 2021-122) or when `RETENTION_MAX_ENTRIES` is `0`.
- build(api): tronc v0.23.0 (was v0.22.1), which brings `mcp.KeepPruned` and v0.22.2's chi v5.5.0 security fixes.
- docs: `docs/gdpr.md` no longer says IP addresses are stored nowhere. Request lines shipped by the collector, Journal's own included, carry `client_ip` and `user_agent` into `log_entries`; the page now lists them, cites GDPR Article 5(1)(e), recital 39 and CNIL deliberation 2021-122, and explains the 90-day choice.

## [0.5.6] - 2026-11-14

### Fixed

- fix(deploy): drop the swarm-only `update_config` and `restart_policy` from the compose services, so `restart: unless-stopped` applies under plain docker compose.

## [0.5.5] - 2026-11-13

### Added

- feat(client): `app.html` loads Vision's tracker (`s.js` with `data-mask`) on `journal.facile.studio` only. `kit.csp` allows the loader by its sha256 hash and `https://vision.facile.studio` in `script-src`; the Go security config adds the same origin to the header policy.

## [0.5.4] - 2026-11-13

### Changed

- build(client): muse v0.21.0 (was v0.18.0). Icon-only controls in muse components now show a muse tooltip.
- fix(client): the log table's app, level and source pivot buttons and the session filter chip show their hint in a muse tooltip instead of the native `title`.

## [0.5.2] - 2026-11-13

### Changed

- chore(deploy): healthchecks run every 30 s with a 1 s start interval, so a deploy cutover stays short while steady `docker exec` load drops.

## [0.5.0] - 2026-11-13

### Changed

- build(api): the schema is ordered goose SQL in `apps/api/migrations/`, embedded in the binary and applied at boot by `tronc/migrate` v0.1.1 behind a Postgres advisory lock. GORM `AutoMigrate` and the idempotent SQL that `schemas.Migrate` ran at every boot are gone; the GORM structs stay as the query layer. A failed migration still exits 1 before the listener binds.
- build(api): `00001_baseline.sql` is production's schema as `pg_dump --schema-only` printed it on 2026-11-13; a database built from it dumps byte-identical to production. A database v0.4.x built (Journal's tables, no applied migration) is stamped at version 1 on its first boot, so the baseline is recorded without running. A database older than v0.4.0 is refused at boot: boot v0.4.3 once, then this release.
- build(api): every DDL statement of the old boot code is in the baseline: porte's identity, session and login-code tables, the generated `search` column and the `log_entries` indexes, the `api_keys` and `alert_rules` defaults and foreign keys, `app_settings`, `directory_deletion_state`, antenne-client's outbox and tronc/mcp's `mcp_actions`. The one-time row conversions no longer run: the local identity re-key, the pre-porte session carry-over, the password adoption, the avatar `NULL` and non-https clean-ups, the `users.created_at` backfill, the `api_keys.kind` and `alert_rules.provider` backfills, and the pre-v1 outbox conversion. Each matched zero rows in production on 2026-11-13, and each can only apply to a database older than v0.4.0, which this release refuses.
- test(api): `TestGORMModelsMatchTheMigratedSchema` fails when a struct names a column the migrations do not build, or a type its field cannot hold. `TestLibrarySchemasAreInTheBaseline` fails when porte, tronc/mcp or antenne-client changes a table Journal keeps its own copy of. `scripts/check.sh` fails on an `.AutoMigrate(` call outside a test file. The tests for the retired conversions are gone with them.
- build(api): goose v3.27.3 raises `segmentio/asm` from v1.1.3 to v1.2.1.

### Added

- `docker exec journal-api /api migrate status` (also `version`, `up` and `baseline <n>`) answers without a shell.
- `00002_timestamp_defaults.sql` gives `created_at` on `alert_rules`, `api_keys`, `log_entries`, `saved_queries` and `source_maps`, and `log_entries.received_at`, a `now()` default, so a row written outside GORM gets a time instead of `NULL`. It has a Down. Rolling back to v0.4.x drops these defaults again, because its `AutoMigrate` removes a default the struct tags do not declare, while the ledger still records version 2: before deploying this release again after such a rollback, run `DELETE FROM goose_db_version WHERE version_id = 2`.

## [0.4.3] - 2026-11-12

### Fixed

- fix(auth): an account re-created for a person Registre erased (a sign-in, a first MCP call or a profile sync racing the erasure) is refused while the subject is in the deletion ledger, and the next poll or push erases any such account again without a restart. porte v0.19.0 (directory v0.5.0): `Deletions.Holds = HoldsIdentity` and `oidc.Deps.Erased`.

## [0.4.2] - 2026-11-11

### Fixed

- fix(brand): favicons, touch and manifest icons, the link preview image and `site.webmanifest` come from muse v0.19.2's brand generator, and `app.html` carries its head block: the catalog title "Journal · Central logs for your infrastructure", its description, theme colours and Open Graph and Twitter tags.
- feat(brand): the sidebar and sign-in screen show the catalog icon `solar:notebook-bold-duotone`, bundled through `registerIcons` so nothing is fetched past the CSP.

## [0.4.1] - 2026-11-11

### Fixed

- fix(deps): vite 7.3.8, @sveltejs/kit 2.70.4 and svelte 5.57.3, plus overrides for devalue 5.9.4, postcss 8.5.32, nanoid 3.3.20, source-map-js 1.2.2 and cookie 0.7.2. Clears every high and medium npm advisory douane reported in the client lockfile.

## [0.4.0] - 2026-11-11

### Added

- feat(api): a remote MCP server at `POST /mcp` on tronc/mcp v0.22.1 and porte/mcpauth v0.2.1, mounted when `MCP_RESOURCE` (compose default `https://journal.facile.studio/mcp`) and `OIDC_ISSUER` are set. Eleven tools: search, read, count and list logs and apps; list, save and delete saved queries; list, create, pause and delete alert rules (administrators only, as in the dashboard). Results are bounded (messages, meta and pages cut under about 25,000 tokens) and webhook URLs never leave the server. Deleting a saved query or an alert rule is confirmed through MRTR elicitation or a link to the new `/pending` page. A person's first MCP call creates their account. See `docs/mcp.md`.
- feat(api): `mcp_actions` table (additive, `CREATE TABLE IF NOT EXISTS`). Erasing an account, through Registre or from Settings, deletes its pending actions in the same transaction.
- feat(client): the Pending actions page at `/pending` (fr/en/es/de), and the sign-in returns to the page that sent the person to it.

### Changed

- build(api): tronc v0.22.1, go-sdk v1.8.0.

## [0.3.3] - 2026-11-11

### Security

- build: Go 1.26.11 in the API, the collector and the SDK (`go` directives, `golang:1.26.11-alpine` builders, mise pin). It fixes GO-2026-6848 to GO-2026-6859 in the standard library and toolchain, which douane reported as unscored gaps and which failed the gate.

## [0.3.2] - 2026-11-11

### Security

- fix(antenne): `GET` and `PUT /api/settings/antenne` no longer return the Antenne shared secret. `settings.secret` becomes `settings.secret_set`, and a `PUT` without `antenne_secret` keeps the stored secret, so the dashboard never holds it. The route stays admin only (`is_admin`).

### Changed

- refactor(api): clear the two findings staticcheck 2026.2.1 reports (S1016, SA4000), which failed `filet gate` before any test ran.

## [0.3.1] - 2026-11-10

### Changed

- build(api): porte v0.18.2. Journal builds its sign-in store with `kit.NewStore`, which reads the `users` table at boot and fails the boot if a column the kit writes is missing. Every column the kit does not write already had a database default, so no account was created with a NULL column; `TestNoUsersColumnIsLeftUnwritten` and `TestAFirstSignInLeavesNoColumnNull` keep it that way.

## [0.3.0] - 2026-11-10

### Added

- feat: the user menu shows pending Registre space invitations. porte/kit's `GET /api/me/invitations` is mounted when `OIDC_ISSUER` is set, and the dashboard passes the answer to muse's `SideBar` and `MobileNav`. No other route moved: a route walk over three configurations (bare, OIDC, OIDC with the CLI audience) lists the same 115 routes with the same unauthenticated status codes, plus this one.
- ci: douane 0.7.0 scans production dependencies on every push (`-fail critical -scope prod`), from a release archive checked against its sha256.
- test(api): `TestIngestContractIsFrozen` pins what every shipper sees on `/api/ingest`, `/api/ingest/browser` and `/api/sourcemaps`: the server key, the browser key, the legacy token, single, batch, empty and gzip payloads, and every status and body.

### Changed

- refactor(api): sign-in runs through `porte/kit`. Its store replaces Journal's own `UserStore`, `kit.RequireAuth` replaces the hand-written middleware, and Registre erasures go through `Store.EraseAccount` in one transaction. One rule changes, by owner decision for every app: the stored email is rewritten at sign-in only when the provider marks it verified. `users.password_hash` and `users.created_at` get database defaults, because the kit inserts only the columns it knows.
- feat(api): `alert.fired` goes through antenne-client v1's outbox. The evaluator marks an Antenne rule fired and queues its event in one transaction, and a drainer sends queued events in id order, so an alert survives a bus outage or a restart. The old, never-written `antenne_outbox` table is converted in place to the library's shape, rows kept. The wire envelope is unchanged. A failed first dial is retried with backoff, and the client reconnects for as long as the process runs. `ANTENNE_URL`, or the older `ANTENNE_INSTANCE`, with `ANTENNE_SECRET`, configures the connection when Settings holds none.
- refactor(api): the four rate limiters are tronc's `middleware.Ratelimit` token buckets with the same rates and keys, per tronc's `docs/ratelimit.md`. A refusal keeps the `rate_limited` envelope; `Retry-After` is now the real wait in whole seconds instead of a fixed 60, and the `X-RateLimit-*` headers are gone. Nothing in the suite read them.
- refactor(client): the dashboard uses `@facile/client` v0.1.0 for its fetch kernel, 401 redirect and theme store. The theme keeps its `journal.theme` key. The browser no longer sends a bearer token from `localStorage`: the session is the HttpOnly cookie, and a leftover `journal.token` is deleted on load.
- build: tronc v0.21.0, porte v0.18.1 (directory v0.4.0), antenne-client `go` v1.0.0 (the `enveloppe` module is gone), muse v0.18.0. The Go modules require go 1.26.10; chi, go-jose, x/crypto and x/text are bumped to clear what douane reports.
- ci: filet v0.19.0 with `preset: facile`. The preset reports 137 findings where the old config reported 63, all of them warnings or information, so the ratchet baseline `.filet-baseline.json` is empty. The pre-push hook also runs `filet gate .`.
- Licence: Journal is now licensed under `AGPL-3.0-only` (see `LICENSE`); before this, the repository carried no licence.
- Licence: the SDKs under `sdk/` (`sdk/browser`, published as `@facile/journal`, and the Go module `sdk/journal`) are `MIT`, so any program may embed them. Each carries its own `LICENSE`. The rest of the repository stays `AGPL-3.0-only`.

## [0.2.4] - 2026-11-07

### Changed

- build(client): bump muse to v0.16.1 and replace the deprecated `outline` button variant with `secondary`.
- build(api): `GET /version` is baked at build time from `apps/client/package.json` by `scripts/version.sh`. The `APP_VERSION` build arg is gone from the Dockerfile, compose file and docs, so a stale Dokploy env can no longer show the wrong version. The response is unchanged.

## [0.2.3] - 2026-11-07

### Changed

- build(client): bump muse to v0.14.4 (from v0.14.2). Switching the language in the user menu re-renders the whole UI at once, with no reload. A single 100% DonutChart slice draws as a full ring.

## [0.2.2] - 2026-11-07

### Security

- Registre tokens are sealed at rest. porte v0.12.1 to v0.13.0: `porte_identities.access_token` and `refresh_token` are encrypted with AES-256-GCM under `PORTE_TOKEN_KEY`. Existing rows are sealed once, in the background, after boot, and the count is logged. An invalid `PORTE_TOKEN_KEY` stops the boot. With the key unset nothing changes.
- Losing `PORTE_TOKEN_KEY` makes the stored tokens unreadable: people sign in again, nothing else is lost. Do not roll back to a build older than porte v0.13.0 after the first reseal.

## [0.2.1] - 2026-11-07

### Fixed

- `GET` and `PUT /api/settings/antenne` use snake_case JSON (`settings.url`, `connected`, `from_env`, `connect_error`) as the dashboard reads it. The Antenne panel and the ingest key list no longer fail with "Cannot read properties of undefined", and saving the Antenne settings sends the `antenne_*` fields the API reads.

### Changed

- The user menu has a Subscription row that links to the Facile spaces page.
- `@facile/muse` v0.14.1 to v0.14.2 (the user menu no longer overflows the viewport edge).

## [0.2.0] - 2026-11-07

### Changed

- The sidebar user card opens the shared muse user menu (account, settings, language, theme, help, sign out). The cog link and the in-settings sign-out are gone.
- Settings use muse `SettingsLayout` with Profile, Preferences, Ingest keys (API, admin only) and Advanced. Profile is read-only and links to Facile; language and theme live in Preferences; deleting the account moved to a `DangerZone` in Advanced. `/settings/appearance` redirects to `/settings/preferences`.
- The ingest keys, Antenne and endpoint sections keep their code and API calls; the key table hides the prefix and created columns below 2xl to fit the narrower panel.
- `@facile/muse` v0.13.0 to v0.14.1.

## [0.1.1] - 2026-11-07

### Changed

- Libraries: porte v0.12.0 to v0.12.1 (GDPR erasure no longer fails on a schema without
  `porte_magic_links`), `@facile/muse` v0.12.1 to v0.13.0 (borderless controls with the light
  `fc-field` fill, `Select` as a custom combobox, badges that never wrap). No app code change:
  the filters and the alert form already use `Select` through `class`, `bind:value` and
  `aria-label`.

## [0.1.0] - 2026-11-07

### Changed

- Libraries: porte v0.5.2 to v0.12.0, porte/directory v0.3.0 (new), tronc v0.16.0 to v0.17.1,
  `@facile/muse` v0.8.0 to v0.12.1.
- Sign-in goes through Registre only. The login page is muse `AuthScreen` with the one
  "Sign in with Facile" button. Accounts that predate the switch keep their data and link by a
  verified address at their first Facile sign-in. An account that already holds another subject
  of the same issuer is never linked (tests cover both).
- tronc v0.17 security headers replace the local middleware: CSP in report-only mode, enforced
  anti-framing, `nosniff`, referrer and permissions policies, HSTS behind a trusted proxy. The
  built client's inline scripts are hashed into `script-src`.
- The whole dashboard is available in French, English, Spanish and German. The session locale
  (the Registre profile's `locale` claim, stored in `users.locale`) drives the language, and
  dates and numbers use `Intl`. The Spanish and German texts need native review.
- The Settings > Advanced page no longer shows a sign-up row; it says sign-in is Registre's.

### Added

- GDPR erasure from Registre: Journal polls `GET /api/directory/deletions` with its OIDC client
  credentials and accepts Registre's signed push on `POST /api/registre/deletions`. A deleted
  person loses their account, identity, sessions and login codes; others are untouched, and a
  repeated notice is a no-op. State is kept in the new `directory_deletion_state` table. Log
  lines that mention the person are not scrubbed: they leave on the `RETENTION_DAYS` clock
  (docs/gdpr.md).
- `GET /version`, from the `APP_VERSION` build argument.
- Migrations (run at boot, idempotent): `users.locale`, `porte_login_codes.provider` and
  `.subject`, `directory_deletion_state`.
- CI: `.github/workflows/ci.yml` runs the API tests against Postgres, the collector, the client
  check, tests and build, the browser SDK tests and filet.

### Removed

- `POST /api/auth/register` and `POST /api/auth/login`, and every password and sign-up screen.
  Ingest keys, the legacy `INGEST_TOKEN` and CLI sign-in keep working.
- `ALLOW_REGISTRATION` and `SSO_ONLY` (compose and config). `GET /api/auth/config` no longer
  returns `allow_registration`.
- The local `SecurityHeaders` middleware and the `.github/workflows/filet.yml` workflow
  (filet now runs from `ci.yml`).
