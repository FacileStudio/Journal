# Journal — GDPR posture

What Journal stores about people, for how long, and how erasure works. This page is the
record of processing for the dashboard itself; it is not legal advice.

## Personal data inventory

| Data | Where | Source |
|---|---|---|
| Email, name, admin flag | `users` | Registration or SSO claims |
| Password (argon2id) | `porte_identities` | Registration / password change |
| Session tokens (SHA-256) | `porte_sessions` | Login; 30-day expiry, named API tokens have none by design |
| Cached avatar files | `AVATAR_DIR` on disk | Downloaded from the identity provider at SSO login |
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

`DELETE /api/auth/me` (dashboard: Settings → Danger zone) deletes the account in one
statement whose WHERE clause also enforces the last-administrator rule, cascading through
every session and identity, removing any locally cached avatar file, expiring the session
cookie, and leaving the caller's token dead.

**Log entries are not touched**: they are keyed by app, never by user, and carry no user
identifier — so there is nothing of the deleted person in them to erase. This is a design
invariant: producers must not put user identifiers into `meta`. The scrubber redacts
credential-shaped keys (`password`, `token`, `cookie`, …) at any depth on both ingest paths,
because the mistake every producer eventually makes is logging a secret by accident.

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
