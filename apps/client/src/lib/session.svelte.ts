import { browser } from '$app/environment';
import { LOCALE_COOKIE, matchLocale, resolveLocale, type Locale } from '@facile/muse';

function readCookie(): string | null {
	const match = document.cookie.split('; ').find((entry) => entry.startsWith(`${LOCALE_COOKIE}=`));
	return match ? decodeURIComponent(match.slice(LOCALE_COOKIE.length + 1)) : null;
}

function initial(): Locale {
	if (!browser) return 'en';
	return matchLocale(readCookie()) ?? resolveLocale(navigator.languages?.join(',') ?? navigator.language);
}

export const session = $state<{ locale: Locale }>({ locale: initial() });

export function rememberLocale(locale: Locale) {
	session.locale = locale;
	if (browser) {
		document.cookie = `${LOCALE_COOKIE}=${locale}; path=/; max-age=31536000; samesite=lax`;
	}
}

export function adoptLocale(locale: Locale) {
	if (browser && readCookie()) return;
	session.locale = locale;
}
