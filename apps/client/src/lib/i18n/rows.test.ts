import { expect, test } from 'bun:test';
import { rows } from './rows';

const placeholders = (text: string) => [...text.matchAll(/\{(\w+)\}/g)].map((match) => match[1]).sort();

test('every key has four non-empty translations with the same placeholders', () => {
	for (const [key, row] of Object.entries(rows)) {
		expect(row.length, key).toBe(4);
		for (const text of row) expect(text.trim().length, key).toBeGreaterThan(0);
		for (const text of row.slice(1)) expect(placeholders(text), key).toEqual(placeholders(row[0]));
	}
});
