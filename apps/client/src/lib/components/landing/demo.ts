import type { LogLevel } from '$lib/backend';

export type DemoRow = { id: number; time: string; app: string; level: LogLevel; message?: string; width: number };

export type TailLine = { time: string; app: string; level: LogLevel; message: string };

export const FILTERS = ['all', 'error', 'warn', 'info'] as const;

export type Filter = (typeof FILTERS)[number];

export const ROWS: DemoRow[] = [
	{ id: 1, time: '14:02:04', app: 'billing', level: 'info', width: 62 },
	{ id: 2, time: '14:02:01', app: 'worker', level: 'error', message: 'thumbnail job crashed', width: 0 },
	{ id: 3, time: '14:01:58', app: 'web', level: 'info', width: 48 },
	{ id: 4, time: '14:01:55', app: 'mailer', level: 'warn', width: 70 },
	{ id: 5, time: '14:01:51', app: 'api', level: 'info', width: 54 },
	{ id: 6, time: '14:01:47', app: 'auth', level: 'error', width: 40 },
	{ id: 7, time: '14:01:44', app: 'billing', level: 'debug', width: 44 },
	{ id: 8, time: '14:01:40', app: 'api', level: 'error', width: 50 },
	{ id: 10, time: '14:01:36', app: 'web', level: 'info', width: 66 },
	{ id: 11, time: '14:01:31', app: 'mailer', level: 'error', width: 46 }
];

export const INCOMING: DemoRow = {
	id: 9,
	time: '14:02:11',
	app: 'worker',
	level: 'error',
	message: 'upload failed: quota exceeded',
	width: 0
};

export const COMMAND = 'journal tail';

export const TAIL: TailLine[] = [
	{ time: '14:02:04', app: 'billing', level: 'info', message: 'nightly export done' },
	{ time: '14:02:06', app: 'web', level: 'info', message: 'cache warmed' },
	{ time: '14:02:08', app: 'mailer', level: 'warn', message: 'smtp retry 2/5' },
	{ time: '14:02:09', app: 'api', level: 'info', message: 'POST /orders 201' },
	{ time: INCOMING.time, app: INCOMING.app, level: INCOMING.level, message: INCOMING.message ?? '' }
];

export function shown(rows: DemoRow[], filter: Filter): DemoRow[] {
	return filter === 'all' ? rows : rows.filter((row) => row.level === filter);
}
