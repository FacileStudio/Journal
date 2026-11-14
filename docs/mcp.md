# MCP server

Journal serves a remote MCP server at `POST /mcp` for any MCP client: Claude (web, Desktop,
Code), ChatGPT, Cursor, VS Code, Bulle. It is free for every account. It runs on
`github.com/FacileStudio/tronc/mcp` v0.22.1 (go-sdk v1.8.0, stateless, spec 2026-07-28), in
`apps/api/modules/mcp`, and is mounted when `MCP_RESOURCE` and `OIDC_ISSUER` are both set.

## Connecting

Add `https://journal.facile.studio/mcp` as a remote MCP server. The client finds Registre
through `/.well-known/oauth-protected-resource/mcp`, signs the person in and asks consent for
`journal.read` and `journal.write`. Tokens are bound to that URL and verified by
`porte/mcpauth`. A token with only `journal.read` gets a 403 `insufficient_scope` challenge on a
write tool. A person's first MCP call creates their Journal account, as a first web sign-in
would.

Registre decides who may get a token at all: Journal's client is `access_policy=assigned`, and
Registre applies the app's policy to its MCP resource at sign-in and on every refresh. That is
the gate that matters, because every Journal account reads every app's logs.

## Access

A tool call sees exactly what the same person sees in the dashboard:

- every account reads the logs, the app list and the saved queries, and may save or delete a
  saved query;
- only an administrator (`users.is_admin`) reads, creates, pauses or deletes alert rules;
- API keys, source map uploads and the Antenne settings have no tool: they hold or mint
  secrets.

## Tools

| Tool | What it does | Hints |
|---|---|---|
| `journal_search_logs` | Entries newest first, by app, level, full text, source, request or session id, and a window (default 24 h) | read-only, open world |
| `journal_read_log` | One entry with its meta, up to 50 neighbours each side, and its stack resolved through source maps | read-only, open world |
| `journal_count_logs` | Per-level counts in up to 90 buckets over at most 90 days | read-only |
| `journal_list_apps` | Apps with their entry count and last entry | read-only |
| `journal_list_saved_queries` | Saved queries with their filters | read-only |
| `journal_save_query` | Save a named filter for everyone | write |
| `journal_delete_saved_query` | Delete a saved query; always confirmed | destructive |
| `journal_list_alerts` | Alert rules, a webhook named by its host only (admin) | read-only |
| `journal_create_alert` | Add an alert rule on a saved query, Antenne by default (admin) | write, open world |
| `journal_set_alert_enabled` | Pause or resume an alert rule (admin) | destructive (pausing silences the alert), idempotent |
| `journal_delete_alert` | Delete an alert rule; always confirmed (admin) | destructive |

`since` and `until` take an RFC 3339 time or a duration back from now (`15m`, `6h`, `7d`).

## Size and sensitive data

Logs can hold personal data and secrets an app logged by mistake, so every result is bounded:

- a message is cut to 1,000 bytes in a list and 16,000 in `journal_read_log`;
- meta appears only in `response_format: detailed` (cut to a 2,000-byte preview per entry) and in
  `journal_read_log` (32,000 bytes);
- a page holds at most 200 entries and is cut further to stay under about 25,000 tokens, with a
  `next_cursor` for the rest;
- a webhook URL never leaves the server: chat and incident webhooks carry their credential in
  the URL.

The server instructions tell the model that log content is data, never instructions.

## Confirmations

Deleting a saved query or an alert rule never runs on the model's word alone:

1. The tool stores a pending action in `mcp_actions` with a digest of the target as it is now.
2. A 2026-07-28 client that declares form elicitation gets an `input_required` question. The
   retry carries the answer and a `requestState` signed with a key derived from
   `OIDC_CLIENT_SECRET`, naming the action, the person and a 15-minute expiry.
3. Any other client gets `status: pending_confirmation` and a `confirm_url`,
   `https://journal.facile.studio/pending?action=<id>`, open for 7 days. The person confirms or
   dismisses it on the Pending actions page; a signed-out person signs in and lands back on it.

Either way the action runs once. A target edited after the question is not deleted and the
action closes. A saved query that an alert rule watches cannot be deleted. A person who lost
administrator rights cannot confirm an alert rule's deletion. Only the person's own browser
session confirms (porte `mcpauth.RequireOwnSession`): a bearer, a delegated (Bulle) token, a CLI
session or a session minted before porte v0.20.0 gets a 403, and the page asks the person to sign
in again in the browser. Every write call and confirmation event is logged without arguments, with
the client id. Erasing an account, through Registre or from Settings, deletes its pending
actions in the same transaction.

## Testing

`go test ./modules/mcp/` with `JOURNAL_TEST_DATABASE_URL` runs tronc's `mcptest` kit
(`TestConformance`) and the tool tests. `MCP_SERVE=127.0.0.1:<port>` with `-run TestServe`
serves the endpoint over a scratch database for MCP Inspector or the official suite
(`npx @modelcontextprotocol/conformance server --url ...`, Node 22).
