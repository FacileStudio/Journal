import { expect, test } from 'bun:test';
import { safePath } from './return-path';

test('only a same-origin path survives the sign-in round trip', () => {
	expect(safePath('/pending?action=abc')).toBe('/pending?action=abc');
	expect(safePath('//evil.example/x')).toBeNull();
	expect(safePath('/\\evil.example')).toBeNull();
	expect(safePath('https://evil.example')).toBeNull();
	expect(safePath('/login?next=/x')).toBeNull();
	expect(safePath('/')).toBeNull();
	expect(safePath(null)).toBeNull();
});
