const KEY = 'journal.next';

/** Keeps only a same-origin path, such as /pending?action=x, never //host or a URL. */
export function safePath(raw: string | null): string | null {
	if (!raw || !raw.startsWith('/') || raw.startsWith('//') || raw.startsWith('/\\')) return null;
	if (raw === '/' || raw.startsWith('/login')) return null;
	return raw;
}

/** Remembers where to go once the Registre sign-in comes back. */
export function rememberReturn(raw: string | null) {
	const path = safePath(raw);
	try {
		if (path) sessionStorage.setItem(KEY, path);
		else sessionStorage.removeItem(KEY);
	} catch {
		return;
	}
}

/** Returns the remembered path once, then forgets it. */
export function takeReturn(): string | null {
	try {
		const path = safePath(sessionStorage.getItem(KEY));
		sessionStorage.removeItem(KEY);
		return path;
	} catch {
		return null;
	}
}
