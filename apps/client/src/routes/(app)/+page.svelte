<script lang="ts">
	import { getContext, onDestroy, onMount } from 'svelte';
	import { backend, type AppSummary, type AuthUser, type HistogramCounts, type HistogramResponse, type ListLogsParams, type LogCursor, type LogEntry, type LogLevel } from '$lib/backend';

	const auth = getContext<{ user: AuthUser | null; logout: () => void }>('auth');

	const levels: LogLevel[] = ['debug', 'info', 'warn', 'error'];
	const MAX_ENTRIES = 2000;
	const CONTEXT_MAX = 200;

	const rangePresets = [
		{ key: '15m', label: '15m', ms: 900_000 },
		{ key: '1h', label: '1h', ms: 3_600_000 },
		{ key: '6h', label: '6h', ms: 21_600_000 },
		{ key: '24h', label: '24h', ms: 86_400_000 },
		{ key: '7d', label: '7d', ms: 604_800_000 },
		{ key: '30d', label: '30d', ms: 2_592_000_000 },
		{ key: 'all', label: 'All', ms: 0 }
	] as const;
	type RangeKey = (typeof rangePresets)[number]['key'] | 'custom';

	type HistBar = { start: number; end: number; counts: HistogramCounts; total: number };
	type Histogram = { bucketSeconds: number; bars: HistBar[]; max: number };

	let apps = $state<AppSummary[]>([]);
	let entries = $state<LogEntry[]>([]);
	let selectedApp = $state<string | null>(null);
	let selectedLevels = $state<LogLevel[]>([]);
	let query = $state('');
	let requestId = $state<string | null>(null);
	let rangePreset = $state<RangeKey>('24h');
	let customSince = $state('');
	let customUntil = $state('');
	let nextBefore = $state<LogCursor | null>(null);
	let loading = $state(false);
	let loadingMore = $state(false);
	let liveTail = $state(false);
	let expandedId = $state<number | null>(null);
	let error = $state('');
	let hist = $state<Histogram | null>(null);

	let contextOpen = $state(false);
	let contextEntries = $state<LogEntry[]>([]);
	let contextAnchorId = $state<number | null>(null);
	let contextBefore = $state(50);
	let contextAfter = $state(50);
	let contextLoading = $state(false);
	let contextError = $state('');

	let searchTimer: ReturnType<typeof setTimeout> | undefined;
	let loadGen = 0;
	let histGen = 0;
	let pollCount = 0;
	let contextId = 0;

	function timeRange(): { since?: string; until?: string } {
		if (rangePreset === 'custom') {
			const range: { since?: string; until?: string } = {};
			if (customSince) range.since = new Date(customSince).toISOString();
			if (customUntil) range.until = new Date(customUntil).toISOString();
			return range;
		}
		const preset = rangePresets.find((entry) => entry.key === rangePreset);
		if (!preset || !preset.ms) return {};
		return { since: new Date(Date.now() - preset.ms).toISOString() };
	}

	function filterParams(): ListLogsParams {
		return {
			app: selectedApp ?? undefined,
			level: selectedLevels.length ? selectedLevels : undefined,
			q: query.trim() || undefined,
			request_id: requestId ?? undefined,
			...timeRange()
		};
	}

	function applyFilters() {
		load();
		loadHistogram();
	}

	async function load() {
		const gen = ++loadGen;
		loading = true;
		error = '';
		try {
			const res = await backend.listLogs({ ...filterParams(), limit: 100 });
			if (gen !== loadGen) return;
			entries = res.entries;
			nextBefore = res.next_before;
		} catch (err) {
			if (gen !== loadGen) return;
			error = err instanceof Error ? err.message : 'Failed to load logs';
		} finally {
			if (gen === loadGen) loading = false;
		}
	}

	async function loadMore() {
		if (nextBefore == null || loadingMore) return;
		const gen = loadGen;
		loadingMore = true;
		try {
			const res = await backend.listLogs({ ...filterParams(), limit: 100, before: nextBefore });
			if (gen !== loadGen) return;
			entries = [...entries, ...res.entries];
			nextBefore = res.next_before;
		} catch (err) {
			if (gen !== loadGen) return;
			error = err instanceof Error ? err.message : 'Failed to load more';
		} finally {
			loadingMore = false;
		}
	}

	async function poll() {
		const gen = loadGen;
		pollCount += 1;
		if (pollCount % 4 === 0) loadHistogram();
		try {
			const res = await backend.listLogs({ ...filterParams(), limit: 100 });
			if (gen !== loadGen || !liveTail) return;
			const maxId = entries.reduce((max, entry) => Math.max(max, entry.id), 0);
			const fresh = res.entries.filter((entry) => entry.id > maxId);
			if (fresh.length) entries = [...fresh, ...entries].slice(0, MAX_ENTRIES);
		} catch {
			/* keep tailing silently */
		}
	}

	async function loadHistogram() {
		const gen = ++histGen;
		const range = timeRange();
		try {
			const res = await backend.histogram(filterParams());
			if (gen !== histGen) return;
			hist = buildHistogram(res, range);
		} catch {
			if (gen === histGen) hist = null;
		}
	}

	function buildHistogram(res: HistogramResponse, range: { since?: string; until?: string }): Histogram {
		const step = res.bucket_seconds * 1000;
		if (!step) return { bucketSeconds: res.bucket_seconds, bars: [], max: 0 };
		const byTime = new Map<number, HistogramCounts>();
		for (const bucket of res.buckets) byTime.set(Date.parse(bucket.ts), bucket.counts);
		const times = [...byTime.keys()];
		const offset = times.length ? ((times[0] % step) + step) % step : 0;
		const rawStart = range.since
			? Date.parse(range.since)
			: times.length
				? Math.min(...times)
				: Date.now() - 86_400_000;
		const rawEnd = range.until ? Date.parse(range.until) : Date.now();
		const start = Math.floor((rawStart - offset) / step) * step + offset;
		const end = Math.max(rawEnd, start + step);
		const count = Math.min(Math.ceil((end - start) / step), 1000);
		const bars: HistBar[] = [];
		let max = 0;
		for (let i = 0; i < count; i++) {
			const ts = start + i * step;
			const counts = byTime.get(ts) ?? {};
			const total = (counts.debug ?? 0) + (counts.info ?? 0) + (counts.warn ?? 0) + (counts.error ?? 0);
			max = Math.max(max, total);
			bars.push({ start: ts, end: ts + step, counts, total });
		}
		return { bucketSeconds: res.bucket_seconds, bars, max };
	}

	function stackSegments(bar: HistBar): { level: LogLevel; y: number; h: number }[] {
		if (!hist || hist.max === 0) return [];
		const segments: { level: LogLevel; y: number; h: number }[] = [];
		let y = 100;
		for (const level of levels) {
			const value = bar.counts[level] ?? 0;
			if (!value) continue;
			const h = (value / hist.max) * 96;
			y -= h;
			segments.push({ level, y, h });
		}
		return segments;
	}

	function bucketTitle(bar: HistBar): string {
		const parts = levels
			.filter((level) => bar.counts[level])
			.map((level) => `${level} ${bar.counts[level]}`);
		return `${formatTime(new Date(bar.start).toISOString())} · ${parts.length ? parts.join(', ') : 'no entries'}`;
	}

	function toLocalInput(ms: number): string {
		const date = new Date(ms);
		const pad = (value: number) => String(value).padStart(2, '0');
		return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
	}

	function zoomToBucket(bar: HistBar) {
		customSince = toLocalInput(Math.floor(bar.start / 60_000) * 60_000);
		customUntil = toLocalInput(Math.max(Math.ceil(bar.end / 60_000) * 60_000, bar.start + 60_000));
		rangePreset = 'custom';
		applyFilters();
	}

	function selectRange(key: RangeKey) {
		rangePreset = key;
		applyFilters();
	}

	function onCustomRange() {
		rangePreset = 'custom';
		applyFilters();
	}

	async function loadApps() {
		try {
			const res = await backend.listApps();
			apps = res.apps;
		} catch {
			apps = [];
		}
	}

	function selectApp(name: string) {
		selectedApp = selectedApp === name ? null : name;
		applyFilters();
	}

	function toggleLevel(level: LogLevel) {
		selectedLevels = selectedLevels.includes(level)
			? selectedLevels.filter((value) => value !== level)
			: [...selectedLevels, level];
		applyFilters();
	}

	function onSearchInput() {
		clearTimeout(searchTimer);
		searchTimer = setTimeout(applyFilters, 300);
	}

	function metaRequestId(entry: LogEntry): string | null {
		const value = entry.meta?.['request_id'];
		if (typeof value === 'string' && value) return value;
		if (typeof value === 'number') return String(value);
		return null;
	}

	function pivotRequest(rid: string) {
		requestId = rid;
		selectedApp = null;
		applyFilters();
	}

	function clearRequestId() {
		requestId = null;
		applyFilters();
	}

	async function openContext(id: number) {
		contextOpen = true;
		contextId = id;
		contextBefore = 50;
		contextAfter = 50;
		contextEntries = [];
		contextAnchorId = null;
		contextError = '';
		await fetchContext();
	}

	async function fetchContext() {
		const id = contextId;
		contextLoading = true;
		contextError = '';
		try {
			const res = await backend.logContext(id, contextBefore, contextAfter);
			if (!contextOpen || id !== contextId) return;
			contextEntries = res.entries;
			contextAnchorId = res.anchor_id;
		} catch (err) {
			if (!contextOpen || id !== contextId) return;
			contextError = err instanceof Error ? err.message : 'Failed to load context';
		} finally {
			if (id === contextId) contextLoading = false;
		}
	}

	function extendContext(direction: 'before' | 'after') {
		if (contextLoading) return;
		if (direction === 'before') contextBefore = Math.min(contextBefore + 50, CONTEXT_MAX);
		else contextAfter = Math.min(contextAfter + 50, CONTEXT_MAX);
		fetchContext();
	}

	function closeContext() {
		contextOpen = false;
	}

	function toggleRow(id: number) {
		expandedId = expandedId === id ? null : id;
	}

	function levelClass(level: LogLevel): string {
		switch (level) {
			case 'error':
				return 'bg-destructive/10 text-destructive';
			case 'warn':
				return 'bg-amber-500/10 text-amber-600';
			case 'debug':
				return 'bg-muted text-muted-foreground';
			default:
				return 'bg-secondary text-secondary-foreground';
		}
	}

	function levelFill(level: LogLevel): string {
		switch (level) {
			case 'error':
				return 'fill-destructive';
			case 'warn':
				return 'fill-amber-600';
			case 'debug':
				return 'fill-muted-foreground';
			default:
				return 'fill-secondary-foreground';
		}
	}

	function formatTime(iso: string): string {
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

	onMount(() => {
		loadApps();
		load();
		loadHistogram();
	});

	onDestroy(() => {
		clearTimeout(searchTimer);
	});

	$effect(() => {
		if (!liveTail) return;
		const interval = setInterval(poll, 2500);
		return () => clearInterval(interval);
	});
</script>

<svelte:head>
	<title>Journal</title>
</svelte:head>

<svelte:window
	onkeydown={(event) => {
		if (event.key === 'Escape' && contextOpen) closeContext();
	}}
/>

<div class="flex h-screen bg-background text-foreground">
	<aside class="hidden w-64 shrink-0 flex-col border-r border-border bg-sidebar md:flex">
		<div class="flex items-center gap-2 border-b border-border px-5 py-4">
			<iconify-icon icon="solar:notebook-bold-duotone" width="22" class="text-foreground"></iconify-icon>
			<span class="text-lg font-bold font-heading tracking-tight">Journal</span>
		</div>

		<div class="flex-1 overflow-y-auto px-3 py-4">
			<div class="mb-1 px-2 text-xs font-medium uppercase tracking-wide text-muted-foreground">Levels</div>
			<div class="mb-5 flex flex-col gap-1">
				{#each levels as level (level)}
					<button
						class="flex items-center justify-between rounded-md px-2 py-1.5 text-sm transition-colors hover:bg-accent {selectedLevels.includes(level) ? 'bg-accent text-accent-foreground' : 'text-muted-foreground'}"
						onclick={() => toggleLevel(level)}
					>
						<span class="inline-flex items-center gap-2">
							<span class="inline-block h-2 w-2 rounded-full {levelClass(level)}"></span>
							{level}
						</span>
						{#if selectedLevels.includes(level)}
							<iconify-icon icon="solar:check-circle-bold" width="16"></iconify-icon>
						{/if}
					</button>
				{/each}
			</div>

			<div class="mb-1 px-2 text-xs font-medium uppercase tracking-wide text-muted-foreground">Time range</div>
			<div class="mb-5 flex flex-col gap-2 px-2">
				<div class="grid grid-cols-4 gap-1">
					{#each rangePresets as preset (preset.key)}
						<button
							class="rounded-md px-1 py-1.5 text-xs transition-colors hover:bg-accent {rangePreset === preset.key ? 'bg-accent font-medium text-accent-foreground' : 'text-muted-foreground'}"
							onclick={() => selectRange(preset.key)}
						>
							{preset.label}
						</button>
					{/each}
				</div>
				<label class="flex flex-col gap-1 text-xs text-muted-foreground">
					From
					<input
						type="datetime-local"
						bind:value={customSince}
						onchange={onCustomRange}
						class="h-9 rounded-md border border-input bg-background px-2 text-xs text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
					/>
				</label>
				<label class="flex flex-col gap-1 text-xs text-muted-foreground">
					To
					<input
						type="datetime-local"
						bind:value={customUntil}
						onchange={onCustomRange}
						class="h-9 rounded-md border border-input bg-background px-2 text-xs text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
					/>
				</label>
			</div>

			<div class="mb-1 px-2 text-xs font-medium uppercase tracking-wide text-muted-foreground">Apps</div>
			<div class="flex flex-col gap-1">
				{#if apps.length === 0}
					<p class="px-2 py-1.5 text-sm text-muted-foreground">No apps yet.</p>
				{:else}
					{#each apps as app (app.name)}
						<button
							class="flex items-center justify-between rounded-md px-2 py-1.5 text-sm transition-colors hover:bg-accent {selectedApp === app.name ? 'bg-accent text-accent-foreground' : 'text-muted-foreground'}"
							onclick={() => selectApp(app.name)}
						>
							<span class="truncate font-medium">{app.name}</span>
							<span class="ml-2 shrink-0 rounded-full bg-muted px-1.5 py-0.5 text-xs text-muted-foreground">{app.count}</span>
						</button>
					{/each}
				{/if}
			</div>
		</div>
	</aside>

	<main class="flex min-w-0 flex-1 flex-col">
		<header class="flex items-center gap-3 border-b border-border px-5 py-3">
			<div class="relative flex-1">
				<iconify-icon icon="solar:magnifer-linear" width="16" class="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground"></iconify-icon>
				<input
					type="search"
					bind:value={query}
					oninput={onSearchInput}
					placeholder="Search messages…"
					class="h-9 w-full rounded-md border border-input bg-background pl-9 pr-3 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
				/>
			</div>
			{#if requestId}
				<span class="inline-flex h-9 max-w-[16rem] items-center rounded-md border border-border bg-accent pl-2 text-xs text-accent-foreground">
					<span class="truncate font-mono" title={requestId}>req:{requestId}</span>
					<button
						class="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-md transition-colors hover:bg-muted"
						aria-label="Clear request filter"
						title="Clear request filter"
						onclick={clearRequestId}
					>
						<iconify-icon icon="solar:close-circle-linear" width="16"></iconify-icon>
					</button>
				</span>
			{/if}
			<button
				class="inline-flex h-9 items-center gap-2 rounded-md border border-border px-3 text-sm font-medium transition-colors {liveTail ? 'bg-primary text-primary-foreground' : 'bg-background hover:bg-accent'}"
				onclick={() => (liveTail = !liveTail)}
			>
				<span class="inline-block h-2 w-2 rounded-full {liveTail ? 'animate-pulse bg-green-500 motion-reduce:animate-none' : 'bg-muted-foreground'}"></span>
				Live tail
			</button>

			<div class="ml-1 flex items-center gap-2 border-l border-border pl-3">
				{#if auth?.user}
					<span class="hidden max-w-[12rem] truncate text-sm text-muted-foreground sm:inline" title={auth.user.email}>
						{auth.user.name || auth.user.email}
					</span>
				{/if}
				{#if auth?.user?.is_admin}
					<a
						href="/keys"
						class="inline-flex h-9 w-9 items-center justify-center rounded-md border border-border bg-background transition-colors hover:bg-accent"
						title="API keys"
						aria-label="API keys"
					>
						<iconify-icon icon="solar:key-linear" width="16"></iconify-icon>
					</a>
				{/if}
				<button
					class="inline-flex h-9 w-9 items-center justify-center rounded-md border border-border bg-background transition-colors hover:bg-accent"
					title="Sign out"
					aria-label="Sign out"
					onclick={() => auth?.logout()}
				>
					<iconify-icon icon="solar:logout-2-linear" width="16"></iconify-icon>
				</button>
			</div>
		</header>

		{#if error}
			<div class="border-b border-border bg-destructive/10 px-5 py-2 text-sm text-destructive">{error}</div>
		{/if}

		{#if hist && hist.bars.length > 0}
			<div class="border-b border-border px-5 pb-2 pt-3">
				<svg
					class="block h-20 w-full"
					viewBox="0 0 {hist.bars.length} 100"
					preserveAspectRatio="none"
					role="img"
					aria-label="Log volume by level over the selected time range"
				>
					{#each hist.bars as bar, i (bar.start)}
						<g
							role="button"
							tabindex="0"
							class="cursor-pointer outline-none"
							onclick={() => zoomToBucket(bar)}
							onkeydown={(event) => {
								if (event.key === 'Enter' || event.key === ' ') {
									event.preventDefault();
									zoomToBucket(bar);
								}
							}}
						>
							<title>{bucketTitle(bar)}</title>
							<rect x={i + 0.08} y="0" width="0.84" height="100" fill="transparent" />
							{#each stackSegments(bar) as segment (segment.level)}
								<rect x={i + 0.08} y={segment.y} width="0.84" height={segment.h} class={levelFill(segment.level)} />
							{/each}
						</g>
					{/each}
				</svg>
				<div class="mt-1 flex justify-between font-mono text-[10px] text-muted-foreground">
					<span>{formatTime(new Date(hist.bars[0].start).toISOString())}</span>
					<span>{formatTime(new Date(hist.bars[hist.bars.length - 1].end).toISOString())}</span>
				</div>
			</div>
		{/if}

		<div class="flex-1 overflow-y-auto">
			<table class="w-full border-collapse text-sm">
				<thead class="sticky top-0 z-10 bg-background">
					<tr class="border-b border-border text-left text-xs uppercase tracking-wide text-muted-foreground">
						<th class="px-5 py-2 font-medium">Time</th>
						<th class="px-3 py-2 font-medium">App</th>
						<th class="px-3 py-2 font-medium">Level</th>
						<th class="px-3 py-2 font-medium">Message</th>
					</tr>
				</thead>
				<tbody>
					{#if loading && entries.length === 0}
						<tr><td colspan="4" class="px-5 py-10 text-center text-muted-foreground">Loading…</td></tr>
					{:else if entries.length === 0}
						<tr><td colspan="4" class="px-5 py-10 text-center text-muted-foreground">No log entries match these filters.</td></tr>
					{:else}
						{#each entries as entry (entry.id)}
							<tr
								class="cursor-pointer border-b border-border/60 align-top transition-colors hover:bg-accent/50"
								onclick={() => toggleRow(entry.id)}
							>
								<td class="whitespace-nowrap px-5 py-2 font-mono text-xs text-muted-foreground">{formatTime(entry.created_at)}</td>
								<td class="px-3 py-2">
									<span class="rounded-md bg-secondary px-1.5 py-0.5 text-xs font-medium text-secondary-foreground">{entry.app}</span>
								</td>
								<td class="px-3 py-2">
									<span class="rounded-md px-1.5 py-0.5 text-xs font-medium uppercase {levelClass(entry.level)}">{entry.level}</span>
								</td>
								<td class="max-w-0 truncate px-3 py-2 font-mono text-xs">{entry.message}</td>
							</tr>
							{#if expandedId === entry.id}
								{@const rid = metaRequestId(entry)}
								<tr class="border-b border-border/60 bg-muted/40">
									<td colspan="4" class="px-5 py-3">
										<div class="mb-2 font-mono text-xs whitespace-pre-wrap break-words">{entry.message}</div>
										{#if entry.meta && Object.keys(entry.meta).length > 0}
											<pre class="overflow-x-auto rounded-md border border-border bg-card p-3 font-mono text-xs">{JSON.stringify(entry.meta, null, 2)}</pre>
										{:else}
											<p class="text-xs text-muted-foreground">No metadata.</p>
										{/if}
										<div class="mt-3 flex flex-wrap items-center gap-2">
											<button
												class="inline-flex h-9 items-center gap-1.5 rounded-md border border-border bg-background px-3 text-xs font-medium transition-colors hover:bg-accent"
												onclick={() => openContext(entry.id)}
											>
												<iconify-icon icon="solar:clock-circle-linear" width="14"></iconify-icon>
												Context
											</button>
											{#if rid}
												<button
													class="inline-flex h-9 items-center gap-1.5 rounded-md border border-border bg-secondary px-3 font-mono text-xs text-secondary-foreground transition-colors hover:bg-accent"
													title="Filter by this request across all apps"
													onclick={() => pivotRequest(rid)}
												>
													<iconify-icon icon="solar:filter-linear" width="14"></iconify-icon>
													request_id: {rid}
												</button>
											{/if}
										</div>
										<div class="mt-2 text-xs text-muted-foreground">received {formatTime(entry.received_at)}</div>
									</td>
								</tr>
							{/if}
						{/each}
					{/if}
				</tbody>
			</table>

			{#if nextBefore != null && entries.length > 0}
				<div class="flex justify-center py-4">
					<button
						class="inline-flex h-9 items-center rounded-md border border-border bg-background px-4 text-sm font-medium transition-colors hover:bg-accent disabled:opacity-50"
						onclick={loadMore}
						disabled={loadingMore}
					>
						{loadingMore ? 'Loading…' : 'Load more'}
					</button>
				</div>
			{/if}
		</div>
	</main>
</div>

{#if contextOpen}
	<div class="fixed inset-0 z-50 flex justify-end">
		<button class="absolute inset-0 bg-foreground/30" aria-label="Close context view" onclick={closeContext}></button>
		<aside class="relative flex h-full w-full max-w-2xl flex-col border-l border-border bg-background shadow-xl">
			<header class="flex items-center justify-between border-b border-border px-5 py-3">
				<h2 class="text-sm font-semibold">Log context</h2>
				<button
					class="inline-flex h-11 w-11 items-center justify-center rounded-md border border-border bg-background transition-colors hover:bg-accent"
					aria-label="Close"
					title="Close"
					onclick={closeContext}
				>
					<iconify-icon icon="solar:close-circle-linear" width="18"></iconify-icon>
				</button>
			</header>

			{#if contextError}
				<div class="border-b border-border bg-destructive/10 px-5 py-2 text-sm text-destructive">{contextError}</div>
			{/if}

			<div class="flex-1 overflow-y-auto">
				<div class="flex justify-center py-3">
					<button
						class="inline-flex h-11 items-center rounded-md border border-border bg-background px-4 text-sm font-medium transition-colors hover:bg-accent disabled:opacity-50"
						onclick={() => extendContext('after')}
						disabled={contextLoading || contextAfter >= CONTEXT_MAX}
					>
						{contextAfter >= CONTEXT_MAX ? 'Newer limit reached' : 'Load newer'}
					</button>
				</div>

				{#if contextLoading && contextEntries.length === 0}
					<p class="px-5 py-8 text-center text-sm text-muted-foreground">Loading…</p>
				{:else if contextEntries.length === 0}
					<p class="px-5 py-8 text-center text-sm text-muted-foreground">No surrounding entries.</p>
				{:else}
					<div class="flex flex-col">
						{#each contextEntries as entry (entry.id)}
							<div class="flex items-start gap-2 border-b border-border/60 px-5 py-2 {entry.id === contextAnchorId ? 'bg-accent ring-1 ring-inset ring-ring' : ''}">
								<span class="whitespace-nowrap font-mono text-xs text-muted-foreground">{formatTime(entry.created_at)}</span>
								<span class="shrink-0 rounded-md bg-secondary px-1.5 py-0.5 text-xs font-medium text-secondary-foreground">{entry.app}</span>
								<span class="shrink-0 rounded-md px-1.5 py-0.5 text-xs font-medium uppercase {levelClass(entry.level)}">{entry.level}</span>
								<span class="min-w-0 flex-1 break-words font-mono text-xs">{entry.message}</span>
							</div>
						{/each}
					</div>
				{/if}

				<div class="flex justify-center py-3">
					<button
						class="inline-flex h-11 items-center rounded-md border border-border bg-background px-4 text-sm font-medium transition-colors hover:bg-accent disabled:opacity-50"
						onclick={() => extendContext('before')}
						disabled={contextLoading || contextBefore >= CONTEXT_MAX}
					>
						{contextBefore >= CONTEXT_MAX ? 'Older limit reached' : 'Load older'}
					</button>
				</div>
			</div>
		</aside>
	</div>
{/if}
