import { chrome } from './rows.chrome';
import { explorer } from './rows.explorer';
import { settings } from './rows.settings';
import { ingest } from './rows.ingest';

export type Row = readonly [en: string, fr: string, es: string, de: string];

export const rows = { ...chrome, ...explorer, ...settings, ...ingest } as const satisfies Record<string, Row>;

export type Key = keyof typeof rows;
