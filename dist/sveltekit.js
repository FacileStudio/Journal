/**
 * SvelteKit glue. Two lines in hooks.client.ts and the app reports.
 *
 * The types are declared structurally rather than imported from
 * @sveltejs/kit, so this package stays dependency-free and version-agnostic:
 * a kit major that only moves types around cannot break an error reporter.
 */
/**
 * Builds a `handleError` for hooks.client.ts.
 *
 * ```ts
 * // src/hooks.client.ts
 * import { createJournal, handleErrorWith } from '@facile/journal';
 *
 * const journal = createJournal({ url: PUBLIC_JOURNAL_URL, key: PUBLIC_JOURNAL_KEY });
 * journal.install();
 * export const handleError = handleErrorWith(journal);
 * ```
 *
 * `install()` covers what escapes to the window; this covers what SvelteKit
 * catches first — a load function that threw, a component that failed to
 * render — which never reaches window.onerror.
 */
export function handleErrorWith(journal, options = {}) {
    const ignore = options.ignoreStatuses ?? [404];
    return ({ error, event, status, message }) => {
        if (!ignore.includes(status)) {
            journal.captureError(error, {
                kind: 'sveltekit',
                route: event?.route?.id ?? undefined,
                url: event?.url?.href,
                meta: { status }
            });
        }
        return { message: options.message ?? message };
    };
}
