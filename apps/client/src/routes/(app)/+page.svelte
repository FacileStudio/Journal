<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { Button, Card, DonutChart, EmptyState, Sparkline, StatCard, Table, icons } from '@facile/muse';
	import { backend, type AppSummary, type LogEntry } from '$lib/backend';
	import { LEVELS, levelFill } from '$lib/levels';
	import { buildHistogram, bucketTotals, type HistBar, type Histogram } from '$lib/histogram';
	import { formatCount, formatRelative, toLocalInput } from '$lib/format';
	import LogHistogram from '$lib/components/LogHistogram.svelte';
	import LogTable from '$lib/components/LogTable.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';

	const WINDOW_MS = 86_400_000;

	let hist = $state<Histogram | null>(null);
	let apps = $state<AppSummary[]>([]);
	let recentErrors = $state<LogEntry[]>([]);
	let loading = $state(true);

	const totals = $derived.by(() => {
		const counts = { debug: 0, info: 0, warn: 0, error: 0 };
		for (const bar of hist?.bars ?? []) {
			for (const level of LEVELS) counts[level] += bar.counts[level] ?? 0;
		}
		return counts;
	});
	const grandTotal = $derived(LEVELS.reduce((sum, level) => sum + totals[level], 0));
	const spark = $derived(hist ? bucketTotals(hist) : []);
	const slices = $derived(
		LEVELS.filter((level) => totals[level] > 0).map((level) => ({
			label: level,
			value: totals[level],
			color: levelFill(level)
		}))
	);
	const topApps = $derived([...apps].sort((a, b) => b.count - a.count).slice(0, 8));

	function openBucket(bar: HistBar) {
		const params = new URLSearchParams({
			range: 'custom',
			since: toLocalInput(Math.floor(bar.start / 60_000) * 60_000),
			until: toLocalInput(Math.max(Math.ceil(bar.end / 60_000) * 60_000, bar.start + 60_000))
		});
		void goto(`/logs?${params}`);
	}

	onMount(async () => {
		const since = new Date(Date.now() - WINDOW_MS).toISOString();
		const [histRes, appsRes, errorsRes] = await Promise.allSettled([
			backend.histogram({ since }),
			backend.listApps(),
			backend.listLogs({ level: ['error'], since, limit: 8 })
		]);
		if (histRes.status === 'fulfilled') hist = buildHistogram(histRes.value, { since });
		if (appsRes.status === 'fulfilled') apps = appsRes.value.apps;
		if (errorsRes.status === 'fulfilled') recentErrors = errorsRes.value.entries;
		loading = false;
	});
</script>

<svelte:head><title>Overview — Journal</title></svelte:head>

<PageHeader title="Overview" description="What every app in the suite has been saying for the last 24 hours.">
	{#snippet actions()}
		<Button size="sm" variant="outline" href="/logs" icon={icons.history}>Open logs</Button>
	{/snippet}
</PageHeader>

<section class="flex flex-col gap-4">
	<div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
		<StatCard label="Entries" value={formatCount(grandTotal)}>
			<Sparkline data={spark} />
		</StatCard>
		<StatCard label="Errors" value={formatCount(totals.error)}>
			<Sparkline data={hist?.bars.map((bar) => bar.counts.error ?? 0) ?? []} color={levelFill('error')} />
		</StatCard>
		<StatCard label="Warnings" value={formatCount(totals.warn)}>
			<Sparkline data={hist?.bars.map((bar) => bar.counts.warn ?? 0) ?? []} color={levelFill('warn')} />
		</StatCard>
		<StatCard label="Apps reporting" value={formatCount(apps.length)} />
	</div>
</section>

<section class="flex flex-col gap-4">
	<div class="flex flex-col gap-1">
		<h2 class="text-fc-lg font-semibold text-fc-fg">Volume</h2>
		<p class="text-fc-sm text-fc-fg-muted">Click a bucket to open it in the log explorer.</p>
	</div>
	<Card>
		<LogHistogram {hist} height={140} onSelect={openBucket} />
	</Card>
</section>

<section class="grid gap-4 lg:grid-cols-2">
	<Card class="flex flex-col gap-4">
		<h2 class="text-fc-lg font-semibold text-fc-fg">By level</h2>
		{#if slices.length > 0}
			<DonutChart
				data={slices}
				class="flex-1"
				centerLabel="entries"
				centerValue={formatCount(grandTotal)}
			/>
		{:else}
			<EmptyState bare icon={icons.dashboard} title="Nothing ingested yet" />
		{/if}
	</Card>

	<Card class="flex flex-col gap-4">
		<h2 class="text-fc-lg font-semibold text-fc-fg">Busiest apps</h2>
		{#if topApps.length > 0}
			<Table class="bg-transparent">
				<thead>
					<tr>
						<th scope="col">App</th>
						<th scope="col">Entries</th>
						<th scope="col">Last seen</th>
					</tr>
				</thead>
				<tbody>
					{#each topApps as app (app.name)}
						<tr>
							<td class="font-fc-mono text-fc-xs">
								<a class="hover:underline" href="/logs?app={encodeURIComponent(app.name)}">{app.name}</a>
							</td>
							<td class="tabular-nums">{formatCount(app.count)}</td>
							<td class="whitespace-nowrap text-fc-fg-muted">{formatRelative(app.last_seen)}</td>
						</tr>
					{/each}
				</tbody>
			</Table>
		{:else}
			<EmptyState bare icon={icons.server} title="No app has shipped a log yet" />
		{/if}
	</Card>
</section>

<section class="flex flex-col gap-4">
	<div class="flex flex-wrap items-end justify-between gap-4">
		<div class="flex flex-col gap-1">
			<h2 class="text-fc-lg font-semibold text-fc-fg">Recent errors</h2>
			<p class="text-fc-sm text-fc-fg-muted">The last eight, across every app.</p>
		</div>
		<Button size="sm" variant="ghost" href="/logs?level=error" iconRight={icons.arrow}>
			See all errors
		</Button>
	</div>
	<LogTable
		entries={recentErrors}
		{loading}
		emptyTitle="No errors in the last 24 hours"
		emptyDescription="Quiet is good. The explorer has the full stream if you want to look closer."
	/>
</section>
