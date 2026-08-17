# @facile/journal — browser SDK

Catches what a page throws and ships it to Journal. Dependency-free, ~4 KB, SvelteKit-shaped
but framework-agnostic.

```sh
bun add github:FacileStudio/Journal#ts
```

## Setup

Mint a **public** key in Journal under **Settings → API** (kind `public`), listing the exact
origins your app is served from. The key ends up in your bundle, which is fine and expected —
it is not a secret. What protects the instance is on the server: the origin allowlist, a
60/min rate limit per key and IP, and a daily quota.

```ts
// src/hooks.client.ts
import { createDeferredJournal } from '@facile/journal';
import { handleErrorWith } from '@facile/journal/sveltekit';

const journal = createDeferredJournal(async () => {
	const config = await fetch('/api/auth/config').then((r) => r.json());
	if (!config.journal?.url || !config.journal?.key) return null;
	return { url: config.journal.url, key: config.journal.key };
});

journal.install();
export const handleError = handleErrorWith(journal);
```

`install()` catches `window.onerror` and `unhandledrejection`. `handleErrorWith` catches what
SvelteKit swallows first — a load function that threw, a component that failed to render —
which never reaches the window. You want both.

### Why the configuration comes over HTTP

Every Facile front is `adapter-static` served by its own Go binary, so the browser has no
environment to read and baking the key into the bundle would make rotating it a rebuild. The
server hands it over instead, on an endpoint the client already calls — porte's `ConfigExtra`
on the Go side:

```go
ConfigExtra: func() map[string]any {
	if appEnv.JournalBrowserKey == "" || appEnv.JournalBrowserURL == "" {
		return nil
	}
	return map[string]any{"journal": map[string]any{
		"url": appEnv.JournalBrowserURL,
		"key": appEnv.JournalBrowserKey,
	}}
},
```

`createDeferredJournal` is what makes that safe: it returns a working client immediately and
buffers what throws while the fetch is in flight — precisely the window boot errors live in. A
loader returning `null`, or throwing, leaves the client permanently inert, so an app with no
key configured goes quiet and an unreachable Journal never becomes an error the page has to
handle. Use `createJournal` directly when the configuration is already in hand.

**The URL must be its own variable, and it must end in `/api`.** `JOURNAL_URL` is the *server*
SDK's and is documented as `http://journal-api:4010` — a Docker-internal address no browser can
resolve, which would give you a page reporting diligently into nowhere. And Journal's dashboard
answers any unmatched path with `200` and an HTML document, so a base URL missing `/api`
discards every report in silence; the SDK warns on the console, and an adopting app should
refuse it at boot.

## API

```ts
createJournal(options);                              // configuration in hand
createDeferredJournal(async () => options | null);   // configuration over HTTP

journal.captureError(error, { level, kind, route, url, meta });
journal.captureMessage('checkout abandoned', { level: 'warn', meta: { step: 3 } });
journal.setUser({ email: 'someone@facile.studio' }); // or null on logout
journal.setContext({ tenant: 'acme' });              // merged into every event
await journal.flush();
```

## Options

| Option | Default | What it does |
|---|---|---|
| `url`, `key` | — | required |
| `release`, `environment` | — | stamped on every event; `release` is what ties an error to a deploy |
| `sampleRate` | `1` | 0–1, applied before queueing |
| `maxEventsPerSession` | `100` | hard stop per page load, so a render loop cannot bill the quota |
| `flushIntervalMs` | `4000` | batches also flush at 20 events and on page hide |
| `ignore` | — | extra `string`/`RegExp` patterns, added to the built-in noise list |
| `beforeSend` | — | last word before an event leaves; return `null` to drop it |
| `user`, `context` | — | initial values for `setUser` / `setContext` |
| `debug` | `false` | logs delivery problems to the console |

## Behaviour worth knowing

- **Every batch carries a session id.** One per tab, kept in `sessionStorage` so a reload stays
  in the same session, landing on the server as `meta.session_id`. Clicking it in the explorer
  is how one error becomes everything else that tab did. Nothing to configure; if storage is
  blocked the id lasts one page load instead.
- **Repeats collapse.** Identical `level + message + top frame` within 60s become one event
  with a `count`. A repeat that arrives after its batch was already sent is dropped until the
  window expires, so `count` is a floor on what happened, never the total.
- **Noise is dropped by default**: `ResizeObserver loop`, `Script error.`, browser-extension
  chatter. Reporting those trains people to ignore the dashboard.
- **Page hide sends a beacon.** A `fetch` started during unload is cancelled, and the
  navigation caused by the error is exactly when the report must not be lost.
- **429 mutes the client** for `Retry-After` seconds. A page that keeps reporting after the
  server said stop turns its own bug into a small outage.
- **A failed delivery is retried** on the next flush, up to 50 queued events. Anything else in
  the 4xx range is dropped — an identical body will not fare better.
- **Nothing throws.** The reporter never adds an exception to code that is already having a
  bad day.
- **The request is `text/plain`**, which keeps it a CORS simple request. `application/json`
  would trigger a preflight, which Journal's app-wide CORS allowlist answers before the route
  is reached — and every report would die there.

## What the server does with it

Entries land in `log_entries` under the key's app, with `meta.source = "browser"`. The server
stamps `origin` and `user_agent` itself — a page cannot claim either — and the client's own
`meta` is scrubbed (`password`, `token`, `cookie`, `authorization` and friends become
`[scrubbed]`), depth-limited and capped at 8 KB.

Filter the explorer on `app = <your app>` and `level = error`, or pivot on `meta.release` to
see what a deploy broke.

## Server-side errors

This package is the browser half. A SvelteKit `hooks.server.ts`, or any Node process, should
use a **secret** key against `POST /api/ingest` — same instance, different route. Go apps use
[`sdk/journal`](../journal), which tees `slog` straight into Journal.

## Development

```sh
bun install
bun test
bun run build     # dist/ is committed; that is what github:FacileStudio/Journal#ts installs
```

`mise run publish-sdk` (`scripts/publish-ts-branch.sh`) rebuilds, checks that `dist/` matches
`src/`, and `git subtree split`s this directory onto the `ts` branch — a `github:` dependency
installs the repository root, so the branch must hold this package at *its* root. `#ts` is a
moving reference, not a version: pushing it is the release.
