import type { LogLevel } from '$lib/backend';

export type DemoRow = { id: number; time: string; app: string; level: LogLevel; message?: string; width: number };

export type TailLine = { time: string; app: string; level: LogLevel; message: string };

export const FILTERS = ['all', 'error', 'warn', 'info'] as const;

export type Filter = (typeof FILTERS)[number];

export const ROWS: DemoRow[] = [
	{ id: 1, time: '14:02:04', app: 'sablier', level: 'info', width: 62 },
	{ id: 2, time: '14:02:01', app: 'nuage', level: 'error', message: 'thumbnail worker crashed', width: 0 },
	{ id: 3, time: '14:01:58', app: 'agenda', level: 'info', width: 48 },
	{ id: 4, time: '14:01:55', app: 'courrier', level: 'warn', width: 70 },
	{ id: 5, time: '14:01:51', app: 'plume', level: 'info', width: 54 },
	{ id: 6, time: '14:01:47', app: 'casier', level: 'error', width: 40 },
	{ id: 7, time: '14:01:44', app: 'sablier', level: 'debug', width: 44 },
	{ id: 8, time: '14:01:40', app: 'plume', level: 'error', width: 50 },
	{ id: 10, time: '14:01:36', app: 'agenda', level: 'info', width: 66 },
	{ id: 11, time: '14:01:31', app: 'courrier', level: 'error', width: 46 }
];

export const INCOMING: DemoRow = {
	id: 9,
	time: '14:02:11',
	app: 'nuage',
	level: 'error',
	message: 'upload failed: quota exceeded',
	width: 0
};

export const COMMAND = 'journal tail';

export const TAIL: TailLine[] = [
	{ time: '14:02:04', app: 'sablier', level: 'info', message: 'timer stopped' },
	{ time: '14:02:06', app: 'agenda', level: 'info', message: 'event synced' },
	{ time: '14:02:08', app: 'courrier', level: 'warn', message: 'smtp retry 2/5' },
	{ time: '14:02:09', app: 'plume', level: 'info', message: 'draft saved' },
	{ time: INCOMING.time, app: INCOMING.app, level: INCOMING.level, message: INCOMING.message ?? '' }
];

export function shown(rows: DemoRow[], filter: Filter): DemoRow[] {
	return filter === 'all' ? rows : rows.filter((row) => row.level === filter);
}
