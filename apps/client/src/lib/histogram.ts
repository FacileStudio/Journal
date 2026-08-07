import type { HistogramCounts, HistogramResponse, LogLevel } from '$lib/backend';
import { LEVELS, levelFill } from '$lib/levels';

export type HistBar = { start: number; end: number; counts: HistogramCounts; total: number };
export type Histogram = { bucketSeconds: number; bars: HistBar[]; max: number };

/** Above this the strip is denser than one bar per pixel and the browser is doing the work twice. */
const MAX_BARS = 1000;

/**
 * The API returns only non-empty buckets. A volume strip has to be continuous or the eye reads
 * a quiet hour as a narrower busy one, so the sparse response is re-expanded onto the bucket
 * grid the server used — `offset` recovers that grid's phase from the first bucket it did send.
 */
export function buildHistogram(
	res: HistogramResponse,
	range: { since?: string; until?: string }
): Histogram | null {
	const step = res.bucket_seconds * 1000;
	if (step === 0) return null;

	const byTime = new Map<number, HistogramCounts>();
	for (const bucket of res.buckets) byTime.set(Date.parse(bucket.ts), bucket.counts);

	const times = [...byTime.keys()].sort((a, b) => a - b);
	const offset = times.length ? ((times[0] % step) + step) % step : 0;

	const rawStart = range.since ? Date.parse(range.since) : (times[0] ?? Date.now() - 86_400_000);
	const rawEnd = range.until ? Date.parse(range.until) : Date.now();
	const start = Math.floor((rawStart - offset) / step) * step + offset;
	const end = Math.max(rawEnd, start + step);
	const count = Math.min(Math.ceil((end - start) / step), MAX_BARS);

	const bars: HistBar[] = [];
	let max = 0;
	for (let i = 0; i < count; i += 1) {
		const barStart = start + i * step;
		const counts = byTime.get(barStart) ?? {};
		const total = LEVELS.reduce((sum, level) => sum + (counts[level] ?? 0), 0);
		if (total > max) max = total;
		bars.push({ start: barStart, end: barStart + step, counts, total });
	}
	return { bucketSeconds: res.bucket_seconds, bars, max };
}

export type Segment = { level: LogLevel; y: number; height: number; fill: string };

/**
 * Stacked geometry in real pixels. Every segment above the baseline one is separated by a 2px
 * gap and rounded on all four corners; the baseline segment keeps its square bottom because it
 * is resting on the axis (CHARTE §12).
 */
export function stackSegments(bar: HistBar, max: number, plotHeight: number): Segment[] {
	if (max <= 0 || bar.total === 0) return [];
	const segments: Segment[] = [];
	let cursor = plotHeight;
	for (const level of LEVELS) {
		const value = bar.counts[level] ?? 0;
		if (!value) continue;
		const height = (value / max) * plotHeight;
		cursor -= height;
		segments.push({ level, y: cursor, height, fill: levelFill(level) });
	}
	return segments.map((segment, index) =>
		index === segments.length - 1 || segment.height <= 3
			? segment
			: { ...segment, height: segment.height - 2 }
	);
}

/** A rect whose bottom corners are square when it rests on the axis. */
export function segmentPath(x: number, y: number, w: number, h: number, baseline: boolean): string {
	const r = Math.max(0, Math.min(2, w / 2, h / 2));
	if (!baseline) {
		return `M${x} ${y + r}a${r} ${r} 0 0 1 ${r}-${r}h${w - 2 * r}a${r} ${r} 0 0 1 ${r} ${r}v${h - 2 * r}a${r} ${r} 0 0 1 -${r} ${r}h-${w - 2 * r}a${r} ${r} 0 0 1 -${r}-${r}z`;
	}
	return `M${x} ${y + r}a${r} ${r} 0 0 1 ${r}-${r}h${w - 2 * r}a${r} ${r} 0 0 1 ${r} ${r}v${h - r}h-${w}z`;
}

export function bucketSummary(bar: HistBar): string {
	const parts = LEVELS.filter((level) => bar.counts[level]).map(
		(level) => `${level} ${bar.counts[level]}`
	);
	return parts.length ? parts.join(', ') : 'no entries';
}

export function bucketTotals(hist: Histogram): number[] {
	return hist.bars.map((bar) => bar.total);
}
