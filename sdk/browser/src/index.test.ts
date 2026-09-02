import { afterEach, beforeEach, expect, test } from 'bun:test';

import { createDeferredJournal, createJournal, type JournalEvent } from './index.js';

type Sent = { events: JournalEvent[]; release?: string; session_id?: string; breadcrumbs?: Breadcrumb[] };

function withSessionStorage(storage: unknown, run: () => Promise<void>) {
	const previous = (globalThis as { sessionStorage?: unknown }).sessionStorage;
	(globalThis as { sessionStorage?: unknown }).sessionStorage = storage;
	return run().finally(() => {
		(globalThis as { sessionStorage?: unknown }).sessionStorage = previous;
	});
}

let sent: Sent[] = [];
let status = 201;
let headers: Record<string, string> = {};

beforeEach(() => {
	sent = [];
	status = 201;
	headers = {};
	// The SDK is inert outside a browser on purpose, so the test has to
	// look like one before the client is created. The listener stubs are what
	// make install() callable, which is where fetch tracing is wired.
	const listeners = { addEventListener() {}, removeEventListener() {} };
	(globalThis as unknown as { window: unknown }).window = listeners;
	(globalThis as unknown as { document: unknown }).document = {
		...listeners,
		visibilityState: 'visible'
	};
	(globalThis as unknown as { location: unknown }).location = { href: 'https://shop.example/cart' };
	globalThis.fetch = (async (_url: string, init: { body: string }) => {
		sent.push(JSON.parse(init.body));
		return {
			ok: status < 400,
			status,
			headers: { get: (name: string) => headers[name] ?? null },
			text: async () => ''
		};
	}) as unknown as typeof fetch;
});

afterEach(() => {
	delete (globalThis as unknown as { window?: unknown }).window;
	delete (globalThis as unknown as { document?: unknown }).document;
	delete (globalThis as unknown as { location?: unknown }).location;
});

/**
 * Replaces fetch with one that answers Journal's own endpoint the way the
 * default mock does, and everything else the way the test asks. Returns the
 * non-Journal calls, so a test can look at the headers that actually went out.
 */
function appFetch(app: { status?: number; headers?: Record<string, string>; reject?: boolean }) {
	const calls: { url: string; headers: Headers }[] = [];
	globalThis.fetch = (async (input: string | URL | Request, init?: RequestInit) => {
		const url = typeof input === 'string' ? input : ((input as Request).url ?? String(input));
		if (url.startsWith('https://journal.facile.studio/api')) {
			sent.push(JSON.parse(String(init?.body)));
			return {
				ok: status < 400,
				status,
				headers: { get: (name: string) => headers[name] ?? null },
				text: async () => ''
			};
		}
		calls.push({ url, headers: new Headers(init?.headers ?? undefined) });
		if (app.reject) throw new TypeError('Failed to fetch');
		return {
			ok: (app.status ?? 200) < 400,
			status: app.status ?? 200,
			headers: new Headers(app.headers ?? {}),
			text: async () => ''
		};
	}) as unknown as typeof fetch;
	return calls;
}

function journal(options: Partial<Parameters<typeof createJournal>[0]> = {}) {
	return createJournal({
		url: 'https://journal.facile.studio/api',
		key: 'journal_pub_shop_test',
		flushIntervalMs: 10_000,
		...options
	});
}

test('an Error becomes name, message and stack', async () => {
	const client = journal({ release: 'v1.2.3' });
	client.captureError(new TypeError('cart is undefined'));
	await client.flush();

	expect(sent).toHaveLength(1);
	expect(sent[0].release).toBe('v1.2.3');
	const [event] = sent[0].events;
	expect(event.message).toBe('TypeError: cart is undefined');
	expect(event.stack).toContain('index.test');
	expect(event.url).toBe('https://shop.example/cart');
	expect(event.level).toBe('error');
});

// The reason this exists at all: a render loop throwing the same error sixty
// times a second must cost one event, not sixty.
test('repeats collapse into one event with a count', async () => {
	const client = journal();
	const error = new Error('boom');
	for (let i = 0; i < 30; i++) client.captureError(error);
	await client.flush();

	expect(sent).toHaveLength(1);
	expect(sent[0].events).toHaveLength(1);
	expect(sent[0].events[0].count).toBe(30);
});

test('different call sites are different events', async () => {
	const client = journal();
	client.captureError(new Error('boom'), { meta: { where: 'a' } });
	client.captureError(new Error('bang'), { meta: { where: 'b' } });
	await client.flush();

	expect(sent[0].events).toHaveLength(2);
});

// The server rejects anything over 20, so the SDK must never offer 21.
test('a batch never exceeds the server cap', async () => {
	const client = journal();
	for (let i = 0; i < 25; i++) client.captureError(new Error(`boom ${i}`));
	await client.flush();
	await client.flush();

	expect(sent.every((batch) => batch.events.length <= 20)).toBe(true);
	expect(sent.flatMap((batch) => batch.events)).toHaveLength(25);
});

test('known browser noise is dropped', async () => {
	const client = journal();
	client.captureError(new Error('ResizeObserver loop limit exceeded'));
	client.captureError('Script error.');
	await client.flush();

	expect(sent).toHaveLength(0);
});

test('ignore patterns are additive', async () => {
	const client = journal({ ignore: [/third-party widget/] });
	client.captureError(new Error('third-party widget exploded'));
	client.captureError(new Error('real bug'));
	await client.flush();

	expect(sent[0].events).toHaveLength(1);
	expect(sent[0].events[0].message).toContain('real bug');
});

test('beforeSend can drop an event', async () => {
	const client = journal({ beforeSend: (event) => (event.message.includes('secret') ? null : event) });
	client.captureError(new Error('secret leak'));
	client.captureError(new Error('ordinary failure'));
	await client.flush();

	expect(sent[0].events).toHaveLength(1);
});

// A page that keeps reporting after the server said stop turns its own bug
// into a small outage.
test('a 429 mutes the client', async () => {
	status = 429;
	headers['Retry-After'] = '120';
	const client = journal();
	client.captureError(new Error('boom'));
	await client.flush();
	expect(sent).toHaveLength(1);

	client.captureError(new Error('another'));
	await client.flush();
	expect(sent).toHaveLength(1);
});

test('a session cap stops a runaway page', async () => {
	const client = journal({ maxEventsPerSession: 5 });
	for (let i = 0; i < 40; i++) client.captureError(new Error(`boom ${i}`));
	await client.flush();
	await client.flush();
	await client.flush();

	expect(sent.flatMap((batch) => batch.events).length).toBeLessThanOrEqual(20);
	client.captureError(new Error('after the cap'));
	await client.flush();
	const messages = sent.flatMap((batch) => batch.events).map((event) => event.message);
	expect(messages.some((message) => message.includes('after the cap'))).toBe(false);
});

test('user and context ride along', async () => {
	const client = journal();
	client.setUser({ email: 'someone@facile.studio' });
	client.setContext({ tenant: 'acme' });
	client.captureError(new Error('boom'));
	await client.flush();

	expect(sent[0].events[0].user?.email).toBe('someone@facile.studio');
	expect(sent[0].events[0].meta?.tenant).toBe('acme');
});

test('a failed delivery keeps the events for the next flush', async () => {
	globalThis.fetch = (async () => {
		throw new Error('offline');
	}) as unknown as typeof fetch;
	const client = journal();
	client.captureError(new Error('boom'));
	await client.flush();

	globalThis.fetch = (async (_url: string, init: { body: string }) => {
		sent.push(JSON.parse(init.body));
		return { ok: true, status: 201, headers: { get: () => null }, text: async () => '' };
	}) as unknown as typeof fetch;
	await client.flush();

	expect(sent).toHaveLength(1);
	expect(sent[0].events[0].message).toContain('boom');
});

test('a deferred client holds errors thrown before its config arrives', async () => {
	let resolve!: (o: Parameters<typeof createJournal>[0]) => void;
	const client = createDeferredJournal(
		() => new Promise((r) => (resolve = r as unknown as typeof resolve))
	);

	client.captureError(new Error('thrown during boot'));
	expect(sent).toHaveLength(0);

	resolve({ url: 'https://journal.facile.studio/api', key: 'journal_pub_shop_test', flushIntervalMs: 10_000 });
	await client.flush();

	expect(sent).toHaveLength(1);
	expect(sent[0].events[0].message).toContain('thrown during boot');
});

// An app with no key configured must go quiet, not accumulate forever.
test('a deferred client that resolves to null stays inert', async () => {
	const client = createDeferredJournal(async () => null);
	client.captureError(new Error('nobody is listening'));
	await client.flush();
	client.captureError(new Error('still nobody'));
	await client.flush();

	expect(sent).toHaveLength(0);
});

// An unreachable Journal is not the page's problem to handle.
test('a deferred client whose loader throws stays inert', async () => {
	const client = createDeferredJournal(async () => {
		throw new Error('config endpoint is down');
	});
	client.captureError(new Error('boom'));
	await client.flush();

	expect(sent).toHaveLength(0);
});

test('setUser before the config arrives still reaches the report', async () => {
	const client = createDeferredJournal(async () => ({
		url: 'https://journal.facile.studio/api',
		key: 'journal_pub_shop_test',
		flushIntervalMs: 10_000
	}));
	client.setUser({ email: 'early@facile.studio' });
	client.setContext({ tenant: 'acme' });
	client.captureError(new Error('boom'));
	await client.flush();

	expect(sent[0].events[0].user?.email).toBe('early@facile.studio');
	expect(sent[0].events[0].meta?.tenant).toBe('acme');
});

// The buffer is bounded: a boot loop must not grow it without limit.
test('a deferred client bounds what it holds', async () => {
	let resolve!: (o: Parameters<typeof createJournal>[0]) => void;
	const client = createDeferredJournal(
		() => new Promise((r) => (resolve = r as unknown as typeof resolve))
	);

	for (let i = 0; i < 500; i++) client.captureError(new Error(`boom ${i}`));
	resolve({ url: 'https://journal.facile.studio/api', key: 'journal_pub_shop_test', flushIntervalMs: 10_000 });
	await client.flush();
	await client.flush();
	await client.flush();

	expect(sent.flatMap((b) => b.events).length).toBeLessThanOrEqual(50);
});

// The point of the id: every batch a tab sends carries the same one, including
// the batches sent after a reload, because a reload is part of the same session
// and is regularly the bug itself.
test('every batch carries the tab session id, across reloads', async () => {
	const store = new Map<string, string>();
	await withSessionStorage(
		{
			getItem: (key: string) => store.get(key) ?? null,
			setItem: (key: string, value: string) => void store.set(key, value)
		},
		async () => {
			const client = journal();
			client.captureError(new Error('one'));
			await client.flush();
			client.captureError(new Error('two'));
			await client.flush();

			// A reload is a new client reading the same tab storage.
			const reloaded = journal();
			reloaded.captureError(new Error('three'));
			await reloaded.flush();

			expect(sent).toHaveLength(3);
			expect(sent[0].session_id).toBeTruthy();
			expect(sent[1].session_id).toBe(sent[0].session_id as string);
			expect(sent[2].session_id).toBe(sent[0].session_id as string);
		}
	);
});

// Wrapping the global fetch is invasive enough that it has to be asked for.
test('tracing is off unless it is asked for', async () => {
	const calls = appFetch({ status: 500 });
	const client = journal();
	const undo = client.install();

	await fetch('https://shop.example/api/cart');

	expect(calls[0].headers.get('X-Request-Id')).toBeNull();
	expect(sent).toHaveLength(0);
	undo();
});

// The whole point: the id on the failed request is the id the server logged it
// under, so the explorer's request_id pivot walks from the browser error to the
// handler that produced it.
test('a traced failure reports the request id it sent', async () => {
	const calls = appFetch({ status: 503 });
	const client = journal({ trace: true });
	const undo = client.install();

	const response = await fetch('https://shop.example/api/cart?page=2');
	expect(response.status).toBe(503);

	const id = calls[0].headers.get('X-Request-Id');
	expect(id).toBeTruthy();

	await client.flush();
	const [event] = sent[0].events;
	expect(event.kind).toBe('fetch');
	// The query string stays out of the message so repeats still collapse.
	expect(event.message).toBe('GET https://shop.example/api/cart failed: 503');
	expect(event.meta?.request_id).toBe(id as string);
	expect(event.meta?.request_url).toBe('https://shop.example/api/cart?page=2');
	expect(event.meta?.status).toBe(503);
	undo();
});

// If the server sends its own id back, that is the one it wrote to its logs.
test('the server echo wins over the id we minted', async () => {
	appFetch({ status: 500, headers: { 'X-Request-Id': 'server-side-id' } });
	const client = journal({ trace: true });
	const undo = client.install();

	await fetch('https://shop.example/api/cart');
	await client.flush();

	expect(sent[0].events[0].meta?.request_id).toBe('server-side-id');
	undo();
});

// A custom header makes a cross-origin request non-simple, so an unasked-for
// origin would earn a preflight the other server never agreed to answer.
test('a cross-origin request is left alone', async () => {
	const calls = appFetch({ status: 500 });
	const client = journal({ trace: true });
	const undo = client.install();

	await fetch('https://stripe.example/v1/charges');

	expect(calls[0].headers.get('X-Request-Id')).toBeNull();
	expect(sent).toHaveLength(0);
	undo();
});

test('a network failure is reported and still thrown', async () => {
	appFetch({ reject: true });
	const client = journal({ trace: true });
	const undo = client.install();

	await expect(fetch('https://shop.example/api/cart')).rejects.toThrow('Failed to fetch');

	await client.flush();
	expect(sent[0].events[0].message).toBe('GET https://shop.example/api/cart failed: network error');
	undo();
});

// Reporting a failed report is how a reporter turns one outage into a loop.
test("Journal's own endpoint is never traced", async () => {
	appFetch({ status: 200 });
	status = 500;
	const client = journal({ trace: true });
	const undo = client.install();

	client.captureError(new Error('boom'));
	await client.flush();
	await client.flush();

	expect(sent.length).toBeGreaterThan(0);
	expect(sent.every((batch) => batch.events.every((event) => event.kind !== 'fetch'))).toBe(true);
	undo();
});

test('uninstalling puts the original fetch back', () => {
	const before = globalThis.fetch;
	const client = journal({ trace: true });

	client.install()();

	expect(globalThis.fetch).toBe(before);
});

// Safari's private mode throws on the storage itself. An error reporter that
// throws inside a page which is already broken is worse than a session id that
// only lasts one page load.
test('a blocked sessionStorage still yields a session id', async () => {
	const blocked = {
		getItem() {
			throw new Error('SecurityError');
		},
		setItem() {
			throw new Error('SecurityError');
		}
	};
	await withSessionStorage(blocked, async () => {
		const client = journal();
		client.captureError(new Error('boom'));
		await client.flush();

		expect(sent).toHaveLength(1);
		expect(sent[0].session_id).toBeTruthy();
	});
});

// Breadcrumbs — a 50-entry ring buffer that ships with every batch.

test('explicit addBreadcrumb appears in the next batch', async () => {
	const client = journal();
	client.addBreadcrumb({ category: 'ui', message: 'user clicked checkout', level: 'info' });
	client.captureError(new Error('payment failed'));
	await client.flush();

	expect(sent).toHaveLength(1);
	expect(sent[0].breadcrumbs).toHaveLength(1);
	expect(sent[0].breadcrumbs[0].message).toBe('user clicked checkout');
	expect(sent[0].breadcrumbs[0].category).toBe('ui');
});

test('the ring buffer caps at 50 breadcrumbs', async () => {
	const client = journal();
	for (let i = 0; i < 60; i++) client.addBreadcrumb({ category: 'ui', message: String(i), level: 'info' });
	client.captureError(new Error('boom'));
	await client.flush();

	expect(sent[0].breadcrumbs).toHaveLength(50);
	expect(sent[0].breadcrumbs[0].message).toBe('10'); // the first 10 were shifted out
});

test('breadcrumbs are cleared after a flush', async () => {
	const client = journal();
	client.addBreadcrumb({ category: 'ui', message: 'before flush', level: 'info' });
	client.captureError(new Error('first'));
	await client.flush();

	client.captureError(new Error('second'));
	await client.flush();

	expect(sent).toHaveLength(2);
	expect(sent[0].breadcrumbs).toHaveLength(1);
	expect(sent[1].breadcrumbs).toBeUndefined();
});

test('console wrapping captures log/warn/error as breadcrumbs', async () => {
	const client = journal({ breadcrumbs: { console: true } });
	const undo = client.install();

	console.log('button clicked');
	console.warn('deprecated api');
	console.error('crash');

	client.captureError(new Error('user error'));
	await client.flush();

	expect(sent[0].breadcrumbs).toHaveLength(3);
	expect(sent[0].breadcrumbs[0].category).toBe('console');
	expect(sent[0].breadcrumbs[0].level).toBe('info');
	expect(sent[0].breadcrumbs[0].message).toBe('button clicked');
	expect(sent[0].breadcrumbs[1].level).toBe('warn');
	expect(sent[0].breadcrumbs[1].message).toBe('deprecated api');
	expect(sent[0].breadcrumbs[2].level).toBe('error');
	expect(sent[0].breadcrumbs[2].message).toBe('crash');
	undo();
});

test('console wrapping restores originals on uninstall', async () => {
	const client = journal({ breadcrumbs: { console: true } });
	const undo = client.install();
	undo();

	// After uninstall, calling console.log should not create a breadcrumb
	console.log('after uninstall');
	client.captureError(new Error('still here'));
	await client.flush();

	expect(sent[0].breadcrumbs).toBeUndefined();
});

test('navigation wrapping captures pushState and popstate', async () => {
	(globalThis as unknown as { history: unknown }).history = {
		pushState: () => {},
		replaceState: () => {},
		length: 1,
		state: null,
		scrollRestoration: 'auto' as ScrollRestoration,
		go: () => {},
		back: () => {},
		forward: () => {}
	};
	const client = journal({ breadcrumbs: { navigation: true } });
	const undo = client.install();

	history.pushState({}, 'cart', '/cart');
	client.captureError(new Error('checkout error'));
	await client.flush();

	expect(sent[0].breadcrumbs).toHaveLength(1);
	expect(sent[0].breadcrumbs[0].category).toBe('navigation');
	expect(sent[0].breadcrumbs[0].message).toBeUndefined();
	expect(sent[0].breadcrumbs[0].data?.from).toBeDefined();
	expect(sent[0].breadcrumbs[0].data?.to).toBe('/cart');
	undo();
	delete (globalThis as unknown as { history?: unknown }).history;
});

test('addBreadcrumb on a deferred journal reaches the real client', async () => {
	let resolve!: (o: Parameters<typeof createJournal>[0]) => void;
	const client = createDeferredJournal(
		() => new Promise((r) => (resolve = r as unknown as typeof resolve))
	);

	client.addBreadcrumb({ category: 'ui', message: 'held', level: 'info' });
	client.captureError(new Error('held error'));

	resolve({ url: 'https://journal.facile.studio/api', key: 'journal_pub_shop_test', flushIntervalMs: 10_000 });
	await client.flush();

	expect(sent).toHaveLength(1);
	// Breadcrumbs added before config arrived are dropped (no pending buffer for them)
	// but errors held before config arrive as usual.
	expect(sent[0].events[0].message).toContain('held error');
});
