import { browser } from '$app/environment';

const TOKEN_KEY = 'journal.token';

let cached: string | null = null;

export function getToken(): string | null {
	if (cached) return cached;
	if (!browser) return null;
	cached = localStorage.getItem(TOKEN_KEY);
	return cached;
}

export function clearToken() {
	cached = null;
	if (browser) localStorage.removeItem(TOKEN_KEY);
}
