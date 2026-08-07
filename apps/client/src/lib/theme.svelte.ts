import { browser } from '$app/environment';

export type ThemeMode = 'system' | 'light' | 'dark';

const KEY = 'journal.theme';

/*
 * `browser`, not a `typeof localStorage === 'undefined'` guard: the SSR runtime defines a
 * `localStorage` global that has no `getItem`, so the typeof check passes and the call
 * still throws.
 */
function stored(): ThemeMode {
	if (!browser) return 'system';
	const raw = localStorage.getItem(KEY);
	return raw === 'light' || raw === 'dark' || raw === 'system' ? raw : 'system';
}

export const theme = $state({ mode: stored() as ThemeMode });

/*
 * Both classes are written, and `system` writes neither. muse's tokens flip on
 * `prefers-color-scheme` scoped to `:root:not(.light)`, so the `.light` class is the only
 * thing that lets someone force light on a dark OS — a script that only ever adds `.dark`
 * strands those users on the media-query palette.
 */
export function setTheme(mode: ThemeMode) {
	theme.mode = mode;
	if (!browser) return;
	const root = document.documentElement;
	root.classList.toggle('dark', mode === 'dark');
	root.classList.toggle('light', mode === 'light');
	localStorage.setItem(KEY, mode);
}

export function applyStoredTheme() {
	setTheme(theme.mode);
}
