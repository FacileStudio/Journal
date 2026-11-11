# Journal — GDPR posture

What Journal stores about people, for how long, and how erasure works. This page is the
record of processing for the dashboard itself; it is not legal advice.

## Personal data inventory

| Data | Where | Source |
|---|---|---|
| Email, name, locale, admin flag, avatar URL | `users` | SSO claims from Registre |
| Legacy password hash (argon2id), no longer usable | `porte_identities`, `users.password_hash` | Accounts created before the switch to Registre |
| Session tokens (SHA-256) | `porte_sessions` | Login; 30-day expiry, named API tokens have none by design |
| Log entries: app, level, message (free text), `meta` (jsonb), origin + user agent for browser events | `log_entries` | The apps shipping logs |

IP addresses are **not stored anywhere** — rate limiting uses them transiently and the quota
table records only key-plus-day counters.

## Retention

- `RETENTION_DAYS` (default **90**): an hourly job deletes `log_entries` older than that many
  days. 90 days is the common defensible window for operational logs under the storage-
  limitation principle; longer windows need a written justification. `RETENTION_DAYS=0` keeps
  entries forever — the boot log warns loudly when it is set.
- Expired sessions are swept hourly regardless of the retention setting. Named API tokens
  have no expiry and are never swept.

## Erasure

Two paths end in the same deletion of the account row, which cascades through every identity
(with its provider tokens), session, named token and login code. Both also delete, in the same
transaction, the person's MCP actions waiting on the confirm page (`mcp_actions`).

- **Self-service.** `DELETE /api/auth/me` (dashboard: Settings → Danger zone) deletes the
  account in one statement whose WHERE clause also enforces the last-administrator rule,
  expires the session cookie and leaves the caller's token dead.
- **Registre erasure.** When Registre erases a person it tells every app. Journal polls
  `GET /api/directory/deletions` with its own OIDC client credentials (every five minutes by
  default) and also accepts Registre's signed push on `POST /api/registre/deletions`. For each
  person it deletes the account behind that Registre subject, then porte erases its own rows.
  The cursor and the list of handled subjects live in `directory_deletion_state`, so a restart
  or a replayed notice does nothing twice, and a failure retries on the next poll. Unlike the
  self-service path, this one never refuses for the last administrator: the person is already
  gone from Registre. It logs a warning when no administrator is left, and an operator then
  promotes another account in the database. Run one replica: the poller assumes it is alone.

Journal keeps no other per-person row. Ingest keys, saved queries, alert rules and source maps
belong to the instance, not to a person, so they stay.

**Log entries are not scrubbed.** Journal does not search log text for a person, and building
that is out of scope. Entries are keyed by app and carry no user identifier of their own, but
any app can write personal data into a message or `meta`. Those lines leave on the retention
clock: `RETENTION_DAYS` (default 90) deletes them, so a person's data in a log line is gone at
most that many days after it was written. Producers must not put user identifiers into `meta`.
The scrubber redacts credential-shaped keys (`password`, `token`, `cookie`, …) at any depth on
both ingest paths, because the mistake every producer eventually makes is logging a secret by
accident.

## Backups

Erasure reaches the live database only. A Postgres dump taken before a deletion still holds
the deleted rows, so:

1. Rotate backups on a schedule **no longer than `RETENTION_DAYS`**, so erased data ages out
   of backups on roughly the same clock as logs do.
2. If an erasure request arrives mid-cycle and cannot wait for rotation, restore-scrub-reseed
   or accept the documented delay — decide before it happens, not during.

## When this instance processes client data

Journal as deployed by Facile holds Facile's own operational logs — controller-only, no DPA
needed. If a client's applications ship logs naming *their* end-users to your instance,
your instance becomes a processor for that data: sign a DPA with the client before the first
log line arrives, and record their retention expectations against `RETENTION_DAYS`.
