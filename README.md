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
import { PUBLIC_JOURNAL_URL, PUBLIC_JOURNAL_KEY } from '$env/static/public';
import { createJournal } from '@facile/journal';
import { handleErrorWith } from '@facile/journal/sveltekit';

const journal = createJournal({
	url: PUBLIC_JOURNAL_URL, // https://journal.facile.studio/api — the /api is load-bearing
	key: PUBLIC_JOURNAL_KEY, // journal_pub_shop_…
	release: __APP_VERSION__,
	environment: 'production'
});

journal.install();
export const handleError = handleErrorWith(journal);
```

`install()` catches `window.onerror` and `unhandledrejection`. `handleErrorWith` catches what
SvelteKit swallows first — a load function that threw, a component that failed to render —
which never reaches the window. You want both.

**`url` must end in `/api`.** Journal's dashboard answers any unmatched path with `200` and an
HTML document, so a base URL without it discards every report in silence. The SDK warns on the
console when it spots this.

## API

```ts
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
