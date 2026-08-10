/**
 * @facile/journal — the browser half of Journal.
 *
 * It catches what a page throws and ships it to POST /ingest/browser, which
 * authenticates with a public key. The key is readable by anyone who opens
 * devtools; the server is what makes that safe, through an origin allowlist
 * and a daily quota. Nothing here is a secret, so nothing here pretends to be.
 */
/** The server's own cap. Sending more is a guaranteed 400. */
const MAX_BATCH = 20;
const DEDUPE_WINDOW_MS = 60000;
const MAX_QUEUE = 50;
/**
 * Noise every page produces and nobody has ever fixed. Reporting it trains
 * people to ignore the dashboard, which costs more than the errors it hides.
 */
// Unanchored on purpose: the message the SDK builds is "Error: …", so a
// pattern pinned to the start of the string would match none of these.
const DEFAULT_IGNORE = [
    /ResizeObserver loop/i,
    /Script error\.?$/i,
    /Java(script)? exception/i,
    /extension context invalidated/i,
    /Non-Error promise rejection captured/i
];
export function createJournal(options) {
    const endpoint = buildEndpoint(options.url, options.key);
    const flushIntervalMs = options.flushIntervalMs ?? 4000;
    const maxEvents = options.maxEventsPerSession ?? 100;
    const sampleRate = options.sampleRate ?? 1;
    const ignore = [...DEFAULT_IGNORE, ...(options.ignore ?? [])];
    let user = options.user ?? null;
    let context = { ...(options.context ?? {}) };
    let queue = [];
    let sent = 0;
    let timer = null;
    let mutedUntil = 0;
    /**
     * The same broken component renders sixty times a second. Collapsing
     * repeats into one event with a count is the difference between a
     * dashboard and a denial of service against your own quota.
     */
    const seen = new Map();
    const enabled = typeof window !== 'undefined';
    function debug(...args) {
        if (options.debug)
            console.warn('[journal]', ...args);
    }
    function schedule() {
        if (timer !== null || queue.length === 0)
            return;
        timer = setTimeout(() => {
            timer = null;
            void flush();
        }, flushIntervalMs);
    }
    function enqueue(event) {
        if (!enabled)
            return;
        if (sent >= maxEvents)
            return;
        if (Date.now() < mutedUntil)
            return;
        if (ignore.some((pattern) => matches(pattern, event.message)))
            return;
        if (sampleRate < 1 && Math.random() >= sampleRate)
            return;
        const prepared = options.beforeSend ? safely(() => options.beforeSend(event), event) : event;
        if (!prepared)
            return;
        const now = Date.now();
        const key = signature(prepared);
        const previous = seen.get(key);
        if (previous && now - previous.at < DEDUPE_WINDOW_MS) {
            // Still queued: the repeat rides out as a count. Already sent:
            // the repeat is dropped until the window expires, so count is a
            // floor on what happened, never the total.
            previous.event.count = (previous.event.count ?? 1) + 1;
            previous.at = now;
            return;
        }
        seen.set(key, { event: prepared, at: now });
        queue.push(prepared);
        if (queue.length > MAX_QUEUE)
            queue = queue.slice(-MAX_QUEUE);
        if (queue.length >= MAX_BATCH)
            void flush();
        else
            schedule();
    }
    function build(error, extra, kind) {
        const described = describe(error);
        return {
            level: extra?.level ?? 'error',
            message: described.message,
            stack: described.stack,
            kind: extra?.kind ?? kind,
            ts: new Date().toISOString(),
            url: extra?.url ?? currentURL(),
            route: extra?.route,
            user: user ?? undefined,
            meta: { ...context, ...(extra?.meta ?? {}) }
        };
    }
    async function send(events, beacon) {
        const body = JSON.stringify({
            release: options.release,
            environment: options.environment,
            events
        });
        // text/plain keeps this a CORS simple request, so no preflight is
        // sent. That is not a micro-optimisation: the preflight would be
        // answered by Journal's app-wide CORS allowlist, which knows nothing
        // about this key's origins, and every report would die there.
        const type = 'text/plain;charset=UTF-8';
        if (beacon && typeof navigator !== 'undefined' && navigator.sendBeacon) {
            return navigator.sendBeacon(endpoint, new Blob([body], { type }));
        }
        const response = await fetch(endpoint, {
            method: 'POST',
            body,
            headers: { 'Content-Type': type },
            mode: 'cors',
            credentials: 'omit',
            keepalive: body.length < 60000
        });
        if (response.status === 429) {
            // Rate limited or out of quota. Going quiet for a while is the
            // only cooperative answer; retrying is how a page turns its own
            // bug into a small outage.
            const retryAfter = Number(response.headers.get('Retry-After') ?? '60');
            mutedUntil = Date.now() + Math.min(Number.isFinite(retryAfter) ? retryAfter : 60, 3600) * 1000;
            debug('rate limited, muted until', new Date(mutedUntil).toISOString());
            return true;
        }
        if (response.status >= 400 && response.status < 500) {
            // The server refused the payload itself. Retrying an identical
            // body cannot help, so drop it and say why once.
            debug('rejected with', response.status, await response.text().catch(() => ''));
            return true;
        }
        return response.ok;
    }
    async function flush(beacon = false) {
        if (!enabled || queue.length === 0)
            return;
        if (timer !== null) {
            clearTimeout(timer);
            timer = null;
        }
        const batch = queue.slice(0, MAX_BATCH);
        queue = queue.slice(batch.length);
        sent += batch.length;
        try {
            const delivered = await send(batch, beacon);
            if (!delivered)
                queue = [...batch, ...queue].slice(0, MAX_QUEUE);
        }
        catch (error) {
            // The network is down or the origin is refused. Keep the events
            // for the next flush and never let the reporter throw into the
            // code that was already having a bad day.
            debug('delivery failed', error);
            queue = [...batch, ...queue].slice(0, MAX_QUEUE);
        }
        schedule();
    }
    function install() {
        if (!enabled)
            return () => { };
        const onError = (event) => {
            enqueue(build(event.error ?? event.message, { url: event.filename || undefined }, 'onerror'));
        };
        const onRejection = (event) => {
            enqueue(build(event.reason, undefined, 'unhandledrejection'));
        };
        // pagehide is the only reliable "this page is going away" signal on
        // iOS Safari, and a navigation caused by the very error being
        // reported is exactly when the queue must not be lost. Both paths
        // send a beacon, because a fetch started during unload is cancelled.
        const onPageHide = () => void flush(true);
        const onVisibilityChange = () => {
            if (document.visibilityState === 'hidden')
                void flush(true);
        };
        window.addEventListener('error', onError);
        window.addEventListener('unhandledrejection', onRejection);
        window.addEventListener('pagehide', onPageHide);
        document.addEventListener('visibilitychange', onVisibilityChange);
        return () => {
            window.removeEventListener('error', onError);
            window.removeEventListener('unhandledrejection', onRejection);
            window.removeEventListener('pagehide', onPageHide);
            document.removeEventListener('visibilitychange', onVisibilityChange);
        };
    }
    return {
        captureError(error, extra) {
            enqueue(build(error, extra, 'captured'));
        },
        captureMessage(message, extra) {
            enqueue(build(message, { level: 'info', ...extra }, 'message'));
        },
        setUser(next) {
            user = next;
        },
        setContext(next) {
            context = { ...context, ...next };
        },
        flush: () => flush(),
        install
    };
}
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
export function createDeferredJournal(load) {
    let real = null;
    let settled = false;
    let installed = false;
    let uninstall = null;
    let user = null;
    let context = {};
    // Bounded, because "waiting for config" is exactly when a boot loop
    // produces thousands of these and nothing is draining them yet.
    const pending = [];
    const ready = (async () => {
        let options = null;
        try {
            options = await load();
        }
        catch {
            options = null;
        }
        settled = true;
        if (!options) {
            pending.length = 0;
            return;
        }
        real = createJournal({ ...options, user: user ?? options.user, context: { ...options.context, ...context } });
        if (installed)
            uninstall = real.install();
        for (const held of pending.splice(0))
            real.captureError(held.error, held.extra);
    })();
    function hold(error, extra) {
        if (settled && !real)
            return;
        if (real) {
            real.captureError(error, extra);
            return;
        }
        if (pending.length < MAX_QUEUE)
            pending.push({ error, extra });
    }
    return {
        captureError: hold,
        captureMessage(message, extra) {
            hold(message, { level: 'info', ...extra });
        },
        setUser(next) {
            user = next;
            real?.setUser(next);
        },
        setContext(next) {
            context = { ...context, ...next };
            real?.setContext(next);
        },
        async flush() {
            await ready;
            await real?.flush();
        },
        install() {
            installed = true;
            if (real)
                uninstall = real.install();
            return () => {
                installed = false;
                uninstall?.();
                uninstall = null;
            };
        }
    };
}
/**
 * A base URL missing /api is the documented way to lose every report in
 * silence: Journal's SPA catch-all answers any unmatched path with 200 and an
 * HTML document, so the SDK would see success forever.
 */
function buildEndpoint(url, key) {
    const base = url.replace(/\/+$/, '');
    if (!base.endsWith('/api')) {
        console.warn(`[journal] url ${url} does not end in /api — Journal's dashboard will answer 200 with HTML and every report will be silently discarded`);
    }
    return `${base}/ingest/browser?key=${encodeURIComponent(key)}`;
}
function describe(error) {
    if (error instanceof Error) {
        const name = error.name || 'Error';
        const message = error.message ? `${name}: ${error.message}` : name;
        const cause = error.cause;
        const stack = error.stack;
        if (cause instanceof Error) {
            return { message, stack: `${stack ?? ''}\ncaused by: ${cause.stack ?? cause.message}` };
        }
        return { message, stack };
    }
    if (typeof error === 'string')
        return { message: error };
    try {
        return { message: `Non-Error thrown: ${JSON.stringify(error)}`.slice(0, 500) };
    }
    catch {
        return { message: 'Non-Error thrown: [unserializable]' };
    }
}
/**
 * The grouping key. The first stack frame is included because two different
 * call sites throwing the same message are two different bugs, and the rest of
 * the stack is left out because it is what varies between them.
 */
function signature(event) {
    const frame = (event.stack ?? '').split('\n')[1]?.trim() ?? '';
    return `${event.level}|${event.message}|${frame}`;
}
function matches(pattern, message) {
    return typeof pattern === 'string' ? message.includes(pattern) : pattern.test(message);
}
function currentURL() {
    return typeof location === 'undefined' ? undefined : location.href;
}
function safely(fn, fallback) {
    try {
        return fn();
    }
    catch {
        return fallback;
    }
}
