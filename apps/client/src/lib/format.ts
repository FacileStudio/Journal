/** Log timestamps: no year, seconds included — the log stream is read at second resolution. */
export function formatTime(iso: string): string {
	const date = new Date(iso);
	if (Number.isNaN(date.getTime())) return iso;
	return date.toLocaleString(undefined, {
		month: 'short',
		day: '2-digit',
		hour: '2-digit',
		minute: '2-digit',
		second: '2-digit',
		hour12: false
	});
}

/** The same instant without its date, for phone widths where the column cannot afford one. */
export function formatClock(iso: string): string {
	const date = new Date(iso);
	if (Number.isNaN(date.getTime())) return iso;
	return date.toLocaleTimeString(undefined, {
		hour: '2-digit',
		minute: '2-digit',
		second: '2-digit',
		hour12: false
	});
}

/** Record timestamps: keys, rules, accounts — dated, minute resolution. */
export function formatDate(iso: string): string {
	const date = new Date(iso);
	if (Number.isNaN(date.getTime())) return iso;
	return date.toLocaleString(undefined, {
		year: 'numeric',
		month: 'short',
		day: '2-digit',
		hour: '2-digit',
		minute: '2-digit',
		hour12: false
	});
}

const RELATIVE_STEPS: [Intl.RelativeTimeFormatUnit, number][] = [
	['second', 60_000],
	['minute', 3_600_000],
	['hour', 86_400_000],
	['day', 2_592_000_000]
];

export function formatRelative(iso: string): string {
	const ms = new Date(iso).getTime();
	if (Number.isNaN(ms)) return iso;
	const delta = ms - Date.now();
	const magnitude = Math.abs(delta);
	const rtf = new Intl.RelativeTimeFormat(undefined, { numeric: 'auto' });
	let divisor = 1000;
	for (const [unit, ceiling] of RELATIVE_STEPS) {
		if (magnitude < ceiling) return rtf.format(Math.round(delta / divisor), unit);
		divisor = ceiling;
	}
	return rtf.format(Math.round(delta / 2_592_000_000), 'month');
}

/** `datetime-local` inputs speak local wall-clock time with no zone, so `toISOString` is wrong. */
export function toLocalInput(ms: number): string {
	const date = new Date(ms);
	const pad = (value: number) => String(value).padStart(2, '0');
	return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

export function formatCount(value: number): string {
	return value.toLocaleString();
}
