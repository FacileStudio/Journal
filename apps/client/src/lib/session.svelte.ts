import { LOCALES, type Locale } from '@facile/muse';
import { rememberLocale as storeLocale, resolveClientLocale } from '@facile/client/locale';

const resolved = resolveClientLocale({ supported: LOCALES });

/** `locale` formats dates and numbers. `hint` is what `provideLocale` takes: null until a source names a locale. */
export const session = $state<{ locale: Locale; hint: Locale | null }>({ locale: resolved.locale, hint: resolved.hint });

/** Applies a locale the person chose and stores it in the suite-wide cookie. */
export function rememberLocale(locale: Locale) {
	session.locale = locale;
	session.hint = locale;
	storeLocale(locale);
}
