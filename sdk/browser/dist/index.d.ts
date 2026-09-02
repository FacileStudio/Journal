/**
 * @facile/journal — the browser half of Journal.
 *
 * It catches what a page throws and ships it to POST /ingest/browser, which
 * authenticates with a public key. The key is readable by anyone who opens
 * devtools; the server is what makes that safe, through an origin allowlist
 * and a daily quota. Nothing here is a secret, so nothing here pretends to be.
 */
export type JournalUser = {
    id?: string;
    email?: string;
};
export type JournalOptions = {
    /** Journal's API base, including the /api suffix. */
    url: string;
    /** A public ingest key: journal_pub_<app>_… */
    key: string;
    /** Build identifier, so an error can be tied to what was deployed. */
    release?: string;
    environment?: string;
    /** 0 to 1. Errors are dropped before they are queued. Default 1. */
    sampleRate?: number;
    /** Hard stop per page load, so a render loop cannot bill the quota. Default 100. */
    maxEventsPerSession?: number;
    flushIntervalMs?: number;
    /** Messages matching any of these are never reported. */
    ignore?: (string | RegExp)[];
    /** Last word before an event leaves. Return null to drop it. */
    beforeSend?: (event: JournalEvent) => JournalEvent | null;
    user?: JournalUser;
    /** Extra meta merged into every event. */
    context?: Record<string, unknown>;
    /**
     * Tie a failed request to the server logs that explain it.
     *
     * `true` traces same-origin requests; an array adds other origins. Off by
     * default, because it wraps the global `fetch` and that is not something a
     * reporter should do without being asked.
     */
    trace?: boolean | string[];
    debug?: boolean;
    breadcrumbs?: {
        console?: boolean;
        navigation?: boolean;
    };
};
export type JournalLevel = 'debug' | 'info' | 'warn' | 'error';
export type JournalEvent = {
    level: JournalLevel;
    message: string;
    ts: string;
    kind?: string;
    stack?: string;
    url?: string;
    route?: string;
    count?: number;
    user?: JournalUser;
    meta?: Record<string, unknown>;
};
export type BreadcrumbCategory = 'console' | 'navigation' | 'ui';
export type Breadcrumb = {
    category: BreadcrumbCategory;
    message?: string;
    level: JournalLevel;
    timestamp: string;
    data?: Record<string, unknown>;
};
export type CaptureExtra = {
    level?: JournalLevel;
    kind?: string;
    route?: string;
    url?: string;
    meta?: Record<string, unknown>;
};
export type Journal = {
    captureError(error: unknown, extra?: CaptureExtra): void;
    captureMessage(message: string, extra?: CaptureExtra): void;
    setUser(user: JournalUser | null): void;
    setContext(context: Record<string, unknown>): void;
    flush(): Promise<void>;
    addBreadcrumb(bc: Omit<Breadcrumb, 'timestamp'>): void;
    /** Wires window.onerror and unhandledrejection. Returns the undo. */
    install(): () => void;
};
export declare function createJournal(options: JournalOptions): Journal;
/**
 * createDeferredJournal returns a working Journal before its configuration
 * exists, and connects it once `load` resolves.
 *
 * Every Facile front is `adapter-static` served by its own Go binary, so there
 * is no runtime environment in the browser and the key arrives from an HTTP
 * call. That call is asynchronous, but `handleError` has to be exported
 * synchronously and the errors worth catching most are the ones that happen
 * during boot — so something has to hold them in the meantime. Doing it here
 * means the fifteen apps that need it do not each invent their own buffer.
 *
 * `load` returning null means "not configured": the client stays inert for the
 * rest of the page's life and drops what it buffered. A load that throws is the
 * same thing, quietly — an unreachable Journal must not become an error the
 * page has to handle.
 */
export declare function createDeferredJournal(load: () => Promise<JournalOptions | null>): Journal;
