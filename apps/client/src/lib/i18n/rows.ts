import type { Row } from '@facile/muse';
import { chrome } from './rows.chrome';
import { explorer } from './rows.explorer';
import { settings } from './rows.settings';
import { ingest } from './rows.ingest';
import { pending } from './rows.pending';
import { landing } from './rows.landing';
import { landingBlocks } from './rows.landingBlocks';
import { spaces } from './rows.spaces';

export const rows = { ...chrome, ...explorer, ...settings, ...ingest, ...pending, ...landing, ...landingBlocks, ...spaces } as const satisfies Record<string, Row>;

export type Key = keyof typeof rows;
