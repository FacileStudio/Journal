import { afterEach, beforeEach, expect, test } from 'bun:test';

import { createDeferredJournal, createJournal, type JournalEvent } from './index.js';

type Sent = { events: JournalEvent[]; release?: string; session_id?: string };

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
	// look like one before the client is created.
	(globalThis as unknown as { window: unknown }).window = {};
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
	delete (globalThis as unknown as { location?: unknown }).location;
});

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
