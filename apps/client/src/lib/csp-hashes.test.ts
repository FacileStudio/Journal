import { createHash } from 'node:crypto';
import { readFileSync } from 'node:fs';
import { expect, test } from 'bun:test';
import config from '../../svelte.config.js';

const html = readFileSync(new URL('../app.html', import.meta.url), 'utf8');

test('every inline script in app.html has its sha256 in the script-src directive', () => {
	const listed = new Set(config.kit?.csp?.directives?.['script-src'] ?? []);
	const bodies = [...html.matchAll(/<script>([\s\S]*?)<\/script>/g)].map((m) => m[1]);
	expect(bodies.length).toBeGreaterThan(0);
	for (const body of bodies) {
		const hash = createHash('sha256').update(body).digest('base64');
		expect(listed.has(`sha256-${hash}`)).toBe(true);
	}
});
