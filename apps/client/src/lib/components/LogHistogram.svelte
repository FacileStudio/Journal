<script lang="ts">
	import { ChartTooltip, resize } from '@facile/muse';
	import { LEVELS, levelFill, levelWeight } from '$lib/levels';
	import { bucketSummary, segmentPath, stackSegments, type HistBar, type Histogram } from '$lib/histogram';
	import { formatTime } from '$lib/format';

	/*
	 * Not a muse `BarChart`: the volume strip is a time *control* as much as a chart — a bucket
	 * is clicked to zoom the whole page into it — and muse's charts expose no selection. The
	 * geometry rules it does follow are the charte's (real pixel dimensions, rounded corners on
	 * every end that faces a gap, aria-hidden drawing beside a hidden data table).
	 */
	let {
		hist,
		height = 96,
		onSelect,
		class: className = ''
	}: {
		hist: Histogram | null;
		height?: number;
		onSelect?: (bar: HistBar) => void;
		class?: string;
	} = $props();

	let width = $state(0);
	let hover = $state(-1);

	const bars = $derived(hist?.bars ?? []);
	const step = $derived(bars.length > 0 && width > 0 ? width / bars.length : 0);
	const barWidth = $derived(Math.max(1, step - Math.min(2, step * 0.2)));
	const tip = $derived(bars[hover] ?? null);

	function rows(bar: HistBar) {
		return LEVELS.filter((level) => bar.counts[level]).map((level) => ({
			name: level,
			value: String(bar.counts[level]),
			color: levelFill(level)
		}));
	}
</script>

<div class={`relative w-full ${className}`} use:resize={(w) => (width = w)}>
	{#if hist && bars.length > 0}
		<svg
			aria-hidden="true"
			class="block"
			{width}
			{height}
			viewBox="0 0 {width} {height}"
			style:height="{height}px"
		>
			{#each bars as bar, i (bar.start)}
				{@const segments = stackSegments(bar, hist.max, height)}
				{#if hover === i}
					<rect
						x={i * step}
						y="0"
						width={step}
						height={height}
						fill="var(--color-fc-surface)"
					/>
				{/if}
				{#each segments as segment, s (segment.level)}
					<path
						d={segmentPath(
							i * step + (step - barWidth) / 2,
							segment.y,
							barWidth,
							segment.height,
							s === segments.length - 1
						)}
						fill={segment.fill}
						fill-opacity={levelWeight(segment.level)}
					/>
				{/each}
			{/each}
		</svg>

		<!-- Real buttons over the drawing: the svg stays decorative, and the server caps the
		     response at 90 buckets so this is never a wall of controls. -->
		<div class="absolute inset-0 flex">
			{#each bars as bar, i (bar.start)}
				<button
					type="button"
					class="h-full min-w-0 flex-1 cursor-pointer focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-fc-ring"
					aria-label="{formatTime(new Date(bar.start).toISOString())} — {bucketSummary(bar)}"
					disabled={!onSelect}
					onmouseenter={() => (hover = i)}
					onfocus={() => (hover = i)}
					onmouseleave={() => (hover = -1)}
					onblur={() => (hover = -1)}
					onclick={() => onSelect?.(bar)}
				></button>
			{/each}
		</div>

		<ChartTooltip
			x={(hover + 0.5) * step}
			y={height / 2}
			title={tip ? formatTime(new Date(tip.start).toISOString()) : ''}
			rows={tip ? rows(tip) : []}
			visible={hover >= 0}
		/>

		<div class="mt-2 flex justify-between font-fc-mono text-fc-xs text-fc-fg-muted">
			<span>{formatTime(new Date(bars[0].start).toISOString())}</span>
			<span>{formatTime(new Date(bars[bars.length - 1].end).toISOString())}</span>
		</div>

		<div class="sr-only">
			<table>
				<caption>Log volume by level over the selected range</caption>
				<thead>
					<tr>
						<th scope="col">Bucket</th>
						{#each LEVELS as level (level)}<th scope="col">{level}</th>{/each}
					</tr>
				</thead>
				<tbody>
					{#each bars.filter((bar) => bar.total > 0) as bar (bar.start)}
						<tr>
							<th scope="row">{formatTime(new Date(bar.start).toISOString())}</th>
							{#each LEVELS as level (level)}<td>{bar.counts[level] ?? 0}</td>{/each}
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{:else}
		<div
			class="flex items-center justify-center text-fc-sm text-fc-fg-muted"
			style:height="{height}px"
		>
			No volume in this range
		</div>
	{/if}
</div>
