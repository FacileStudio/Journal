/**
 * SvelteKit glue. Two lines in hooks.client.ts and the app reports.
 *
 * The types are declared structurally rather than imported from
 * @sveltejs/kit, so this package stays dependency-free and version-agnostic:
 * a kit major that only moves types around cannot break an error reporter.
 */
import type { Journal } from './index.js';
type ClientErrorInput = {
    error: unknown;
    event: {
        url?: URL;
        route?: {
            id?: string | null;
        };
    };
    status: number;
    message: string;
};
export type HandleErrorOptions = {
    /**
     * Statuses to leave alone. 404 is the default because a bad URL is a
     * visitor's typo, not a defect, and reporting it buries the real errors.
     */
    ignoreStatuses?: number[];
    /** What the user sees. SvelteKit renders this on the error page. */
    message?: string;
};
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
export declare function handleErrorWith(journal: Journal, options?: HandleErrorOptions): ({ error, event, status, message }: ClientErrorInput) => {
    message: string;
};
export {};
