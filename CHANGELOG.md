# Changelog

## [Unreleased]

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
