import type { LogLevel } from '$lib/backend';

export const LEVELS: LogLevel[] = ['debug', 'info', 'warn', 'error'];

/*
 * Log severity is exactly what muse's tone vocabulary was built for, so a level never gets a
 * colour of its own — it borrows the status token that already means the same thing. `debug`
 * is deliberately toneless: it is the level you read past.
 */
export type LevelTone = 'neutral' | 'info' | 'warning' | 'danger';

const TONES: Record<LogLevel, LevelTone> = {
	debug: 'neutral',
	info: 'info',
	warn: 'warning',
	error: 'danger'
};

const FILLS: Record<LogLevel, string> = {
	debug: 'var(--color-fc-fg-muted)',
	info: 'var(--color-fc-info)',
	warn: 'var(--color-fc-warning)',
	error: 'var(--color-fc-danger)'
};

export function levelTone(level: LogLevel): LevelTone {
	return TONES[level] ?? 'neutral';
}

/*
 * Written out rather than interpolated: Tailwind only emits a utility it has literally seen in
 * a source file, so `bg-fc-${tone}/10` compiles to nothing at all.
 */
const CHIPS: Record<LogLevel, string> = {
	debug: 'bg-fc-surface text-fc-fg-muted',
	info: 'bg-fc-info/10 text-fc-info',
	warn: 'bg-fc-warning/10 text-fc-warning',
	error: 'bg-fc-danger/10 text-fc-danger'
};

export function levelChipClass(level: LogLevel): string {
	return CHIPS[level] ?? CHIPS.debug;
}

/** SVG `fill` / chart `color`, which take a paint value rather than a Tailwind class. */
export function levelFill(level: LogLevel): string {
	return FILLS[level] ?? FILLS.debug;
}

/*
 * Volume-strip emphasis. At suite ratios `info` is ~60% of every bucket, and painted at full
 * strength it turns the strip into a wall of blue that hides exactly the two levels anyone is
 * scanning for. Hue still matches the row badges — only the weight changes.
 */
const WEIGHTS: Record<LogLevel, number> = {
	debug: 0.3,
	info: 0.45,
	warn: 0.9,
	error: 1
};

export function levelWeight(level: LogLevel): number {
	return WEIGHTS[level] ?? 1;
}

export function isLevel(value: string): value is LogLevel {
	return (LEVELS as string[]).includes(value);
}
