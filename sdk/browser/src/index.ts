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
	message: string;
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

/** The server's own cap. Sending more is a guaranteed 400. */
const MAX_BATCH = 20;
const DEDUPE_WINDOW_MS = 60_000;
const MAX_QUEUE = 50;
const MAX_BREADCRUMBS = 50;
const SESSION_KEY = 'journal.session';

/**
 * Noise every page produces and nobody has ever fixed. Reporting it trains
 * people to ignore the dashboard, which costs more than the errors it hides.
 */
// Unanchored on purpose: the message the SDK builds is "Error: …", so a
// pattern pinned to the start of the string would match none of these.
const DEFAULT_IGNORE: RegExp[] = [
	/ResizeObserver loop/i,
	/Script error\.?$/i,
	/Java(script)? exception/i,
	/extension context invalidated/i,
	/Non-Error promise rejection captured/i
];

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

export function createJournal(options: JournalOptions): Journal {
	const endpoint = buildEndpoint(options.url, options.key);
	const sessionId = sessionIdentifier();
	/* Journal's own API is never traced: reporting a failed report is how a
	   reporter turns one outage into a loop. */
	const apiBase = options.url.replace(/\/+$/, '');
	const flushIntervalMs = options.flushIntervalMs ?? 4000;
	const maxEvents = options.maxEventsPerSession ?? 100;
	const sampleRate = options.sampleRate ?? 1;
	const ignore = [...DEFAULT_IGNORE, ...(options.ignore ?? [])];
	/* With no base url there is no endpoint to recognise, so the wrapper could not
	   tell the reporter's own requests from the app's and would report its own
	   failures in a loop. Tracing nothing is the safe reading of a misconfigured
	   client; buildEndpoint has already warned about it. */
	const traced = apiBase ? tracedOrigins(options.trace) : null;

	let user = options.user ?? null;
	let context = { ...(options.context ?? {}) };
	let queue: JournalEvent[] = [];
	let sent = 0;
	let timer: ReturnType<typeof setTimeout> | null = null;
	let mutedUntil = 0;
	let breadcrumbs: Breadcrumb[] = [];

	/** Ring buffer for breadcrumbs — max 50 entries. */
	function pushBreadcrumb(bc: Breadcrumb) {
		breadcrumbs.push(bc);
		if (breadcrumbs.length > MAX_BREADCRUMBS) breadcrumbs = breadcrumbs.slice(-MAX_BREADCRUMBS);
	}
	function takeBreadcrumbs(): Breadcrumb[] {
		const batch = breadcrumbs;
		breadcrumbs = [];
		return batch;
	}

	/**
	 * The same broken component renders sixty times a second. Collapsing
	 * repeats into one event with a count is the difference between a
	 * dashboard and a denial of service against your own quota.
	 */
	const seen = new Map<string, { event: JournalEvent; at: number }>();

	const enabled = typeof window !== 'undefined';

	function debug(...args: unknown[]) {
		if (options.debug) console.warn('[journal]', ...args);
	}

	function schedule() {
		if (timer !== null || queue.length === 0) return;
		timer = setTimeout(() => {
			timer = null;
			void flush();
		}, flushIntervalMs);
	}

	function enqueue(event: JournalEvent) {
		if (!enabled) return;
		if (sent >= maxEvents) return;
		if (Date.now() < mutedUntil) return;
		if (ignore.some((pattern) => matches(pattern, event.message))) return;
		if (sampleRate < 1 && Math.random() >= sampleRate) return;

		const prepared = options.beforeSend ? safely(() => options.beforeSend!(event), event) : event;
		if (!prepared) return;

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
		if (queue.length > MAX_QUEUE) queue = queue.slice(-MAX_QUEUE);
		if (queue.length >= MAX_BATCH) void flush();
		else schedule();
	}

	function build(error: unknown, extra: CaptureExtra | undefined, kind: string): JournalEvent {
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

	async function send(events: JournalEvent[], beacon: boolean, breadcrumbsToSend: Breadcrumb[]): Promise<boolean> {
		const body = JSON.stringify({
			release: options.release,
			environment: options.environment,
			session_id: sessionId,
			breadcrumbs: breadcrumbsToSend.length > 0 ? breadcrumbsToSend : undefined,
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
			keepalive: body.length < 60_000
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

	async function flush(beacon = false): Promise<void> {
		if (!enabled || queue.length === 0) return;
		if (timer !== null) {
			clearTimeout(timer);
			timer = null;
		}

		const batch = queue.slice(0, MAX_BATCH);
		queue = queue.slice(batch.length);
		sent += batch.length;

		const breadcrumbBatch = takeBreadcrumbs();

		try {
			const delivered = await send(batch, beacon, breadcrumbBatch);
			if (!delivered) queue = [...batch, ...queue].slice(0, MAX_QUEUE);
		} catch (error) {
			// The network is down or the origin is refused. Keep the events
			// for the next flush and never let the reporter throw into the
			// code that was already having a bad day.
			debug('delivery failed', error);
			queue = [...batch, ...queue].slice(0, MAX_QUEUE);
		}
		schedule();
	}

	/**
	 * Wraps `fetch` so a failed request carries the id the server logged it
	 * under. Journal already holds both halves of the suite's logs, so once a
	 * browser event has a `request_id` the existing pivot walks straight from
	 * the error to the handler that produced it.
	 *
	 * Only failures become events. A 4xx is usually the application working —
	 * an expired session, a 404 probe — and reporting those trains people to
	 * ignore the dashboard, which is the one thing this package must not do.
	 */
	function instrument(): () => void {
		if (!traced || typeof globalThis.fetch !== 'function') return () => {};

		const original = globalThis.fetch;
		if ((original as { [TRACED]?: boolean })[TRACED]) return () => {};

		const wrapped = async (input: RequestInfo | URL, init?: RequestInit): Promise<Response> => {
			let target: string | null = null;
			let id: string | undefined;
			let nextInit = init;
			try {
				target = requestTarget(input);
				if (target && allowedByTrace(target, traced) && !target.startsWith(apiBase)) {
					id = randomId();
					nextInit = withRequestID(input, init, id);
				} else {
					target = null;
				}
			} catch {
				/* Instrumentation is never worth a failed request. Whatever went
				   wrong here, the call still goes out exactly as it was. */
				target = null;
				id = undefined;
				nextInit = init;
			}

			if (!target) return original(input, init);

			const method = requestMethod(input, init);
			try {
				const response = await original(input, nextInit);
				if (response.status >= 500) {
					report(method, target, response.status, headerId(response) ?? id);
				}
				return response;
			} catch (error) {
				report(method, target, 0, id);
				throw error;
			}
		};

		(wrapped as { [TRACED]?: boolean })[TRACED] = true;
		globalThis.fetch = wrapped as typeof fetch;
		return () => {
			if (globalThis.fetch === (wrapped as typeof fetch)) globalThis.fetch = original;
		};
	}

	/**
	 * The request URL keeps its query string in meta and loses it in the
	 * message, so a hundred failures against the same endpoint collapse into one
	 * event with a count instead of a hundred unique ones.
	 */
	function report(method: string, url: string, status: number, id: string | undefined) {
		const outcome = status === 0 ? 'network error' : String(status);
		enqueue({
			level: 'error',
			message: `${method} ${withoutQuery(url)} failed: ${outcome}`,
			kind: 'fetch',
			ts: new Date().toISOString(),
			url: currentURL(),
			user: user ?? undefined,
			meta: {
				...context,
				request_url: url,
				request_method: method,
				...(status > 0 ? { status } : {}),
				...(id ? { request_id: id } : {})
			}
		});
	}

	function formatConsoleArgs(args: unknown[]): string {
		return args
			.slice(0, 3)
			.map((arg) => {
				const text = typeof arg === 'string' ? arg : safeStringify(arg);
				return text.length > 200 ? `${text.slice(0, 200)}…` : text;
			})
			.join(' ');
	}
	function safeStringify(value: unknown): string {
		try {
			return JSON.stringify(value);
		} catch {
			return String(value);
		}
	}

	function wrapConsole(): () => void {
		const orig: { log: typeof console.log; warn: typeof console.warn; error: typeof console.error } = {
			log: console.log.bind(console),
			warn: console.warn.bind(console),
			error: console.error.bind(console)
		};
		console.log = (...args: unknown[]) => {
			orig.log(...args);
			pushBreadcrumb({ category: 'console', message: formatConsoleArgs(args), level: 'info', timestamp: new Date().toISOString() });
		};
		console.warn = (...args: unknown[]) => {
			orig.warn(...args);
			pushBreadcrumb({ category: 'console', message: formatConsoleArgs(args), level: 'warn', timestamp: new Date().toISOString() });
		};
		console.error = (...args: unknown[]) => {
			orig.error(...args);
			pushBreadcrumb({ category: 'console', message: formatConsoleArgs(args), level: 'error', timestamp: new Date().toISOString() });
		};
		return () => {
			console.log = orig.log;
			console.warn = orig.warn;
			console.error = orig.error;
		};
	}

	function wrapNavigation(): () => void {
		const origPush = history.pushState.bind(history);
		const origReplace = history.replaceState.bind(history);
		history.pushState = (data, title, url) => {
			const result = origPush(data, title, url);
			pushBreadcrumb({ category: 'navigation', message: `pushState: ${title || ''}`, level: 'info', timestamp: new Date().toISOString(), data: { url: String(url ?? '') } });
			return result;
		};
		history.replaceState = (data, title, url) => {
			const result = origReplace(data, title, url);
			pushBreadcrumb({ category: 'navigation', message: `replaceState: ${title || ''}`, level: 'info', timestamp: new Date().toISOString(), data: { url: String(url ?? '') } });
			return result;
		};
		const onPop = () => {
			pushBreadcrumb({ category: 'navigation', message: 'popstate', level: 'info', timestamp: new Date().toISOString() });
		};
		window.addEventListener('popstate', onPop);
		return () => {
			history.pushState = origPush;
			history.replaceState = origReplace;
			window.removeEventListener('popstate', onPop);
		};
	}

	function install(): () => void {
		if (!enabled) return () => {};

		const onError = (event: ErrorEvent) => {
			enqueue(build(event.error ?? event.message, { url: event.filename || undefined }, 'onerror'));
		};
		const onRejection = (event: PromiseRejectionEvent) => {
			enqueue(build(event.reason, undefined, 'unhandledrejection'));
		};
		// pagehide is the only reliable "this page is going away" signal on
		// iOS Safari, and a navigation caused by the very error being
		// reported is exactly when the queue must not be lost. Both paths
		// send a beacon, because a fetch started during unload is cancelled.
		const onPageHide = () => void flush(true);
		const onVisibilityChange = () => {
			if (document.visibilityState === 'hidden') void flush(true);
		};

		window.addEventListener('error', onError);
		window.addEventListener('unhandledrejection', onRejection);
		window.addEventListener('pagehide', onPageHide);
		document.addEventListener('visibilitychange', onVisibilityChange);
		const uninstrument = instrument();

		const unrestoreBreadcrumbs: (() => void)[] = [];
		if (options.breadcrumbs?.console && typeof console !== 'undefined') {
			unrestoreBreadcrumbs.push(wrapConsole());
		}
		if (options.breadcrumbs?.navigation && typeof history !== 'undefined') {
			unrestoreBreadcrumbs.push(wrapNavigation());
		}

		return () => {
			window.removeEventListener('error', onError);
			window.removeEventListener('unhandledrejection', onRejection);
			window.removeEventListener('pagehide', onPageHide);
			document.removeEventListener('visibilitychange', onVisibilityChange);
			uninstrument();
			for (const restore of unrestoreBreadcrumbs) restore();
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
		install,
		addBreadcrumb(bc: Omit<Breadcrumb, 'timestamp'>) {
			pushBreadcrumb({ ...bc, timestamp: new Date().toISOString() });
		}
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
export function createDeferredJournal(load: () => Promise<JournalOptions | null>): Journal {
	let real: Journal | null = null;
	let settled = false;
	let installed = false;
	let uninstall: (() => void) | null = null;
	let user: JournalUser | null = null;
	let context: Record<string, unknown> = {};

	// Bounded, because "waiting for config" is exactly when a boot loop
	// produces thousands of these and nothing is draining them yet.
	const pending: { error: unknown; extra?: CaptureExtra }[] = [];

	const ready = (async () => {
		let options: JournalOptions | null = null;
		try {
			options = await load();
		} catch {
			options = null;
		}
		settled = true;
		if (!options) {
			pending.length = 0;
			return;
		}

		real = createJournal({ ...options, user: user ?? options.user, context: { ...options.context, ...context } });
		if (installed) uninstall = real.install();
		for (const held of pending.splice(0)) real.captureError(held.error, held.extra);
	})();

	function hold(error: unknown, extra?: CaptureExtra) {
		if (settled && !real) return;
		if (real) {
			real.captureError(error, extra);
			return;
		}
		if (pending.length < MAX_QUEUE) pending.push({ error, extra });
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
			if (real) uninstall = real.install();
			return () => {
				installed = false;
				uninstall?.();
				uninstall = null;
			};
		},
		addBreadcrumb(bc: Omit<Breadcrumb, 'timestamp'>) {
			real?.addBreadcrumb(bc);
		}
	};
}

/**
 * A base URL missing /api is the documented way to lose every report in
 * silence: Journal's SPA catch-all answers any unmatched path with 200 and an
 * HTML document, so the SDK would see success forever.
 */
function buildEndpoint(url: string, key: string): string {
	const base = url.replace(/\/+$/, '');
	if (!base.endsWith('/api')) {
		console.warn(
			`[journal] url ${url} does not end in /api — Journal's dashboard will answer 200 with HTML and every report will be silently discarded`
		);
	}
	return `${base}/ingest/browser?key=${encodeURIComponent(key)}`;
}

function describe(error: unknown): { message: string; stack?: string } {
	if (error instanceof Error) {
		const name = error.name || 'Error';
		const message = error.message ? `${name}: ${error.message}` : name;
		const cause = (error as { cause?: unknown }).cause;
		const stack = error.stack;
		if (cause instanceof Error) {
			return { message, stack: `${stack ?? ''}\ncaused by: ${cause.stack ?? cause.message}` };
		}
		return { message, stack };
	}
	if (typeof error === 'string') return { message: error };
	try {
		return { message: `Non-Error thrown: ${JSON.stringify(error)}`.slice(0, 500) };
	} catch {
		return { message: 'Non-Error thrown: [unserializable]' };
	}
}

/**
 * The grouping key. The first stack frame is included because two different
 * call sites throwing the same message are two different bugs, and the rest of
 * the stack is left out because it is what varies between them.
 */
function signature(event: JournalEvent): string {
	const frame = (event.stack ?? '').split('\n')[1]?.trim() ?? '';
	return `${event.level}|${event.message}|${frame}`;
}

function matches(pattern: string | RegExp, message: string): boolean {
	return typeof pattern === 'string' ? message.includes(pattern) : pattern.test(message);
}

function currentURL(): string | undefined {
	return typeof location === 'undefined' ? undefined : location.href;
}

/**
 * The tab's identity, stable across reloads.
 *
 * sessionStorage rather than a module constant, because a reload belongs to the
 * same debugging session — often it *is* the bug — and a per-page id would cut
 * the story in half. It dies with the tab, which is the scope the word session
 * is supposed to mean.
 *
 * Both halves have a fallback: storage throws outright in Safari's private
 * mode, and `crypto.randomUUID` does not exist outside a secure context. An id
 * that is merely unlikely to collide beats a reporter that throws inside a page
 * which is already having a bad day.
 */
function sessionIdentifier(): string {
	try {
		const existing = sessionStorage.getItem(SESSION_KEY);
		if (existing) return existing;
		const fresh = randomId();
		sessionStorage.setItem(SESSION_KEY, fresh);
		return fresh;
	} catch {
		return randomId();
	}
}

/** Marks a wrapped fetch, so two clients on one page do not stack wrappers. */
const TRACED = Symbol.for('journal.traced');

/**
 * Resolves the trace option into the origins allowed to receive the header.
 *
 * Same-origin is the default and the safe one: a custom header makes a
 * cross-origin request non-simple, so it earns a preflight the other server has
 * to answer. Naming an origin here is opting into that, and into teaching that
 * server to allow `X-Request-Id`.
 */
function tracedOrigins(option: boolean | string[] | undefined): string[] | null {
	if (!option) return null;
	const own = typeof location === 'undefined' ? [] : [new URL(location.href).origin];
	if (option === true) return own;
	return [...own, ...option.map((origin) => origin.replace(/\/+$/, ''))];
}

function allowedByTrace(url: string, origins: string[]): boolean {
	try {
		return origins.includes(new URL(url).origin);
	} catch {
		return false;
	}
}

/** Absolute, because a relative URL cannot be matched against an origin. */
function requestTarget(input: RequestInfo | URL): string | null {
	const raw =
		typeof input === 'string'
			? input
			: input instanceof URL
				? input.href
				: ((input as Request)?.url ?? null);
	if (!raw) return null;
	return typeof location === 'undefined' ? raw : new URL(raw, location.href).href;
}

function requestMethod(input: RequestInfo | URL, init?: RequestInit): string {
	const method = init?.method ?? (typeof input === 'object' && 'method' in input ? input.method : '');
	return (method || 'GET').toUpperCase();
}

/**
 * Returns the init to call with, carrying the id.
 *
 * A bare `Request` is mutated in place: rebuilding one would clone a body that
 * may already be locked, and losing the request is a far worse bug than losing
 * the correlation. Everything else gets a copied init, so the caller's own
 * object is never edited underneath it.
 */
function withRequestID(
	input: RequestInfo | URL,
	init: RequestInit | undefined,
	id: string
): RequestInit | undefined {
	const carriesHeaders = typeof input === 'object' && 'headers' in input;
	if (carriesHeaders && !init) {
		(input as Request).headers.set('X-Request-Id', id);
		return init;
	}
	const next: RequestInit = { ...(init ?? {}) };
	const headers = new Headers(
		next.headers ?? (carriesHeaders ? (input as Request).headers : undefined)
	);
	headers.set('X-Request-Id', id);
	next.headers = headers;
	return next;
}

/**
 * The server's own id wins when it sends one back: whatever it logged under is
 * the value that will match. Reading it cross-origin needs
 * `Access-Control-Expose-Headers`, so cross-origin falls back to ours.
 */
function headerId(response: Response): string | undefined {
	try {
		return response.headers.get('X-Request-Id') ?? undefined;
	} catch {
		return undefined;
	}
}

function withoutQuery(url: string): string {
	const cut = url.indexOf('?');
	return cut === -1 ? url : url.slice(0, cut);
}

function randomId(): string {
	return typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function'
		? crypto.randomUUID()
		: `${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 10)}`;
}

function safely<T>(fn: () => T, fallback: T): T {
	try {
		return fn();
	} catch {
		return fallback;
	}
}
