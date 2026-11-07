import { createTranslator, getI18n, type Catalogs, type Locale, type Translator } from '@facile/muse';
import { rows, type Key } from './rows';

export type { Key };

type Catalog = Record<Key, string>;

const LOCALE_INDEX: Record<Locale, number> = { en: 0, fr: 1, es: 2, de: 3 };

function column(locale: Locale): Catalog {
	const index = LOCALE_INDEX[locale];
	const out = {} as Catalog;
	for (const key of Object.keys(rows) as Key[]) out[key] = rows[key][index];
	return out;
}

export const catalogs: Catalogs<Catalog> = {
	en: column('en'),
	fr: column('fr'),
	es: column('es'),
	de: column('de')
};

export function useT(): Translator<Catalog> {
	const i18n = getI18n();
	return createTranslator(catalogs, () => i18n.locale);
}
