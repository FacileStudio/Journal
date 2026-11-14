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
| Client IP address, user agent and path of HTTP requests | `log_entries`, inside request lines | The suite apps' request logs, Journal's own included, shipped by the collector |
| Open and recently resolved MCP confirmations (user id, action summary) | `mcp_actions` | MCP tool calls that need a confirmation |

Journal's own tables store no IP address: rate limiting uses it transiently and the quota table
records only key-plus-day counters. Log entries do. The suite's request logger writes
`client_ip`, `remote_addr` and `user_agent` on every request line, and the collector ships
those lines into `log_entries`. GDPR recital 30 names IP addresses among the online identifiers
that can identify a person, so these lines are personal data and leave on the retention clock
below like every other entry.

## Retention

One job (`internal/retention`) enforces two bounds at boot and then hourly, deleting in
batches of 10,000 rows.

- **Rolling window.** `RETENTION_DAYS` (default **90**) deletes `log_entries` whose
  `created_at` is older than that many days. This is GDPR storage limitation, Article 5(1)(e):
  data is kept "for no longer than is necessary for the purposes for which the personal data
  are processed", and recital 39 asks the controller to set "time limits ... for erasure or for
  a periodic review". `created_at` is the producer's timestamp, but ingest replaces one more
  than five minutes in the future with the receipt time, so no entry can stamp itself out of
  the window.
- **Ceiling.** `RETENTION_MAX_ENTRIES` (default **10,000,000**) keeps at most that many
  entries and deletes the oldest received first. It protects the disk, not a legal limit: on
  2026-11-14 production took in about 800,000 entries a day at about 1.1 KB each on disk, so
  a 90-day window alone would hold about 72 million rows and 80 GB. When the ceiling bites, the
  real window is shorter than `RETENTION_DAYS`, which only helps storage limitation.

Why 90 days. Journal holds operational logs: errors, request lines and browser events, read to
debug and to spot incidents. CNIL deliberation 2021-122 recommends six months to a year for the
traceability logs of an application, the record of who created, read, changed or deleted
personal data, and asks for a documented justification beyond a year. Journal is not that
record for the suite's apps, so the CNIL range is an upper reference here, not a target. 90
days covers incident investigation and the alert windows with room to spare.

Guard rails in the boot log:

- `RETENTION_DAYS=0` keeps entries forever and warns: the CNIL security guide lists keeping
  traces "sans limite de durée" among what not to do.
- `RETENTION_DAYS` above 365 warns, citing CNIL deliberation 2021-122.
- `RETENTION_MAX_ENTRIES=0` warns that the table then grows with ingest volume.

Other rows with a clock:

- Expired sessions are swept hourly regardless of the retention setting. Named API tokens
  have no expiry and are never swept.
- `mcp_actions` rows resolved or expired more than 30 days ago are deleted hourly by tronc's
  `mcp.KeepPruned`.
- `antenne_outbox` rows are deleted by `antenne-client` seven days after they are sent.

Sources: GDPR text, https://eur-lex.europa.eu/legal-content/EN/TXT/HTML/?uri=CELEX:32016R0679 ;
CNIL deliberation 2021-122, https://www.cnil.fr/sites/cnil/files/atoms/files/recommandation_-_journalisation.pdf ;
CNIL security guide, sheet "Tracer les opérations", https://www.cnil.fr/fr/securite-tracer-les-operations

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
most that many days after it was written, or sooner when the ceiling removes it. Producers
must not put user identifiers into `meta`.
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
