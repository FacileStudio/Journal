<script lang="ts">
	import { onDestroy, onMount } from 'svelte';
	import { page } from '$app/state';
	import { replaceState } from '$app/navigation';
	import {
		Alert,
		Button,
		Card,
		Input,
		Modal,
		Select,
		StatusDot,
		Tabs,
		icons,
		toast
	} from '@facile/muse';
	import {
		backend,
		type AppSummary,
		type ListLogsParams,
		type LogCursor,
		type LogEntry,
		type LogLevel,
		type SavedQuery,
		type SavedQueryParams
	} from '$lib/backend';
	import { LEVELS, isLevel, levelChipClass } from '$lib/levels';
	import { buildHistogram, type HistBar, type Histogram } from '$lib/histogram';
	import { toLocalInput } from '$lib/format';
	import LogHistogram from '$lib/components/LogHistogram.svelte';
	import LogTable from '$lib/components/LogTable.svelte';
	import LogContextDrawer from '$lib/components/LogContextDrawer.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';

	const MAX_ENTRIES = 2000;
	const PAGE_SIZE = 100;
	const POLL_MS = 2500;

	const RANGES = [
		{ id: '15m', label: '15m', ms: 900_000 },
		{ id: '1h', label: '1h', ms: 3_600_000 },
		{ id: '6h', label: '6h', ms: 21_600_000 },
		{ id: '24h', label: '24h', ms: 86_400_000 },
		{ id: '7d', label: '7d', ms: 604_800_000 },
		{ id: '30d', label: '30d', ms: 2_592_000_000 },
		{ id: 'all', label: 'All', ms: 0 },
		{ id: 'custom', label: 'Custom', ms: 0 }
	];

	let apps = $state<AppSummary[]>([]);
	let savedQueries = $state<SavedQuery[]>([]);
	let entries = $state<LogEntry[]>([]);
	let hist = $state<Histogram | null>(null);

	let selectedApp = $state('');
	let selectedLevels = $state<LogLevel[]>([]);
	let query = $state('');
	let requestId = $state('');
	let sessionId = $state('');
	let source = $state('');
	let range = $state('24h');
	let customSince = $state('');
	let customUntil = $state('');

	let nextBefore = $state<LogCursor | null>(null);
	let loading = $state(false);
	let loadingMore = $state(false);
	let error = $state('');
	let liveTail = $state(false);
	let tailPaused = $state(false);
	let gapIds = $state<number[]>([]);
	let contextAnchor = $state<number | null>(null);

	let saveOpen = $state(false);
	let saveName = $state('');
	let saving = $state(false);

	/*
	 * Generation counters, not AbortController: a stale response is discarded rather than
	 * cancelled, so a filter change mid-flight can never repaint the table with the old filter's
	 * rows. `pollCount` refreshes the histogram every fourth tick — ten seconds, not 2.5.
	 */
	let loadGen = 0;
	let pollCount = 0;
	let searchTimer: ReturnType<typeof setTimeout> | undefined;

	const activeCount = $derived(
		(selectedApp ? 1 : 0) +
		selectedLevels.length +
		(query.trim() ? 1 : 0) +
		(source ? 1 : 0) +
		(requestId ? 1 : 0) +
		(sessionId ? 1 : 0)
	);

	function timeRange(): { since?: string; until?: string } {
		if (range === 'custom') {
			const out: { since?: string; until?: string } = {};
			if (customSince) out.since = new Date(customSince).toISOString();
			if (customUntil) out.until = new Date(customUntil).toISOString();
			return out;
		}
		const preset = RANGES.find((entry) => entry.id === range);
		if (!preset || !preset.ms) return {};
		return { since: new Date(Date.now() - preset.ms).toISOString() };
	}

	function filterParams(): ListLogsParams {
		return {
			app: selectedApp || undefined,
			level: selectedLevels.length ? selectedLevels : undefined,
			q: query.trim() || undefined,
			source: source || undefined,
			request_id: requestId || undefined,
			session_id: sessionId || undefined,
			...timeRange()
		};
	}

	function syncUrl() {
		const params = new URLSearchParams();
		if (selectedApp) params.set('app', selectedApp);
		if (selectedLevels.length) params.set('level', selectedLevels.join(','));
		if (query.trim()) params.set('q', query.trim());
		if (source) params.set('source', source);
		if (requestId) params.set('request_id', requestId);
		if (sessionId) params.set('session_id', sessionId);
		if (range !== '24h') params.set('range', range);
		if (range === 'custom') {
			if (customSince) params.set('since', customSince);
			if (customUntil) params.set('until', customUntil);
		}
		const search = params.toString();
		replaceState(search ? `?${search}` : page.url.pathname, {});
	}

	function readUrl() {
		const params = page.url.searchParams;
		selectedApp = params.get('app') ?? '';
		selectedLevels = (params.get('level') ?? '').split(',').filter(isLevel);
		query = params.get('q') ?? '';
		source = params.get('source') ?? '';
		requestId = params.get('request_id') ?? '';
		sessionId = params.get('session_id') ?? '';
		const wanted = params.get('range');
		range = wanted && RANGES.some((entry) => entry.id === wanted) ? wanted : '24h';
		customSince = params.get('since') ?? '';
		customUntil = params.get('until') ?? '';
	}

	function apply() {
		syncUrl();
		void load();
		void loadHistogram();
	}

	async function load() {
		const gen = ++loadGen;
		loading = true;
		error = '';
		try {
			const res = await backend.listLogs({ ...filterParams(), limit: PAGE_SIZE });
			if (gen !== loadGen) return;
			entries = res.entries;
			nextBefore = res.next_before;
			gapIds = [];
		} catch (err) {
			if (gen !== loadGen) return;
			error = err instanceof Error ? err.message : 'Failed to load logs';
		} finally {
			if (gen === loadGen) loading = false;
		}
	}

	async function loadMore() {
		if (!nextBefore || loadingMore) return;
		const gen = loadGen;
		loadingMore = true;
		try {
			const res = await backend.listLogs({
				...filterParams(),
				limit: PAGE_SIZE,
				before: nextBefore
			});
			if (gen !== loadGen) return;
			entries = [...entries, ...res.entries];
			nextBefore = res.next_before;
		} catch (err) {
			if (gen !== loadGen) return;
			error = err instanceof Error ? err.message : 'Failed to load more';
		} finally {
			if (gen === loadGen) loadingMore = false;
		}
	}

	async function loadHistogram() {
		const bounds = timeRange();
		try {
			const res = await backend.histogram(filterParams());
			hist = buildHistogram(res, bounds);
		} catch {
			hist = null;
		}
	}

	async function poll() {
		const gen = loadGen;
		pollCount += 1;
		if (pollCount % 4 === 0) void loadHistogram();
		try {
			const res = await backend.listLogs({ ...filterParams(), limit: PAGE_SIZE });
			if (gen !== loadGen || !liveTail || tailPaused) return;
			const maxId = entries.reduce((max, entry) => Math.max(max, entry.id), 0);
			const fresh = res.entries.filter((entry) => entry.id > maxId);
			if (!fresh.length) return;
			/* A full page of new rows means the tail could not keep up — mark where the stream
			   was cut so nobody reads two adjacent rows as consecutive. */
			if (fresh.length === PAGE_SIZE && entries.length > 0) {
				gapIds = [...gapIds, fresh[fresh.length - 1].id];
			}
			entries = [...fresh, ...entries].slice(0, MAX_ENTRIES);
		} catch {
			/* keep tailing silently — a blip should not tear down the view */
		}
	}

	async function loadApps() {
		try {
			apps = (await backend.listApps()).apps;
		} catch {
			apps = [];
		}
	}

	async function loadSavedQueries() {
		try {
			savedQueries = (await backend.listQueries()).queries;
		} catch {
			savedQueries = [];
		}
	}

	onMount(() => {
		readUrl();
		void loadApps();
		void loadSavedQueries();
		void load();
		void loadHistogram();
	});

	onDestroy(() => clearTimeout(searchTimer));

	$effect(() => {
		if (!liveTail || tailPaused) return;
		const timer = setInterval(() => void poll(), POLL_MS);
		return () => clearInterval(timer);
	});

	function onSearchInput() {
		clearTimeout(searchTimer);
		searchTimer = setTimeout(apply, 300);
	}

	function toggleLevel(level: LogLevel) {
		selectedLevels = selectedLevels.includes(level)
			? selectedLevels.filter((entry) => entry !== level)
			: [...selectedLevels, level];
		apply();
	}

	function pivotApp(app: string) {
		selectedApp = selectedApp === app ? '' : app;
		apply();
	}

	/* Browser entries carry meta.source; nothing else does. Clicking it is what
	   makes the client-error view reachable without knowing the schema. */
	function pivotSource(value: string) {
		source = source === value ? '' : value;
		apply();
	}

	function pivotRequest(id: string) {
		requestId = id;
		/* A request crosses services, so pinning it to one app would hide most of the trace. */
		selectedApp = '';
		apply();
	}

	/* A session is one tab, and one tab can hit several fronts, so the app filter
	   goes with it. The level filter stays: the error that led here is usually
	   not the only thing worth seeing, but the reader asked for a level. */
	function pivotSession(id: string) {
		sessionId = id;
		selectedApp = '';
		apply();
	}

	function clearFilters() {
		selectedApp = '';
		selectedLevels = [];
		query = '';
		source = '';
		requestId = '';
		sessionId = '';
		apply();
	}

	function zoomTo(bar: HistBar) {
		customSince = toLocalInput(Math.floor(bar.start / 60_000) * 60_000);
		customUntil = toLocalInput(
			Math.max(Math.ceil(bar.end / 60_000) * 60_000, bar.start + 60_000)
		);
		range = 'custom';
		apply();
	}

	function applySaved(id: string) {
		const saved = savedQueries.find((entry) => String(entry.id) === id);
		if (!saved) return;
		selectedApp = saved.params.app ?? '';
		selectedLevels = (saved.params.levels ?? []).filter(isLevel);
		query = saved.params.q ?? '';
		source = saved.params.source ?? '';
		requestId = saved.params.request_id ?? '';
		apply();
	}

	async function saveQuery() {
		const name = saveName.trim();
		if (!name) return;
		saving = true;
		try {
			const params: SavedQueryParams = {};
			if (selectedApp) params.app = selectedApp;
			if (selectedLevels.length) params.levels = selectedLevels;
			if (query.trim()) params.q = query.trim();
			if (source) params.source = source;
			if (requestId) params.request_id = requestId;
			await backend.createQuery(name, params);
			await loadSavedQueries();
			saveOpen = false;
			saveName = '';
			toast.success(`Saved “${name}”.`);
		} catch (err) {
			toast.danger(err instanceof Error ? err.message : 'Could not save the query');
		} finally {
			saving = false;
		}
	}

	function toggleTail() {
		liveTail = !liveTail;
		tailPaused = false;
		if (liveTail) apply();
	}

	function toggleTailPause() {
		tailPaused = !tailPaused;
		if (!tailPaused) apply();
	}
</script>

<svelte:head><title>Logs — Journal</title></svelte:head>

<PageHeader title="Logs" description="Every entry shipped to this instance, newest first.">
	{#snippet actions()}
		{#if liveTail}
			<Button
				size="sm"
				variant="outline"
				icon={tailPaused ? icons.refresh : icons.clock}
				onclick={toggleTailPause}
			>
				{tailPaused ? 'Resume' : 'Pause'}
			</Button>
		{/if}
		<Button size="sm" variant={liveTail ? 'primary' : 'outline'} onclick={toggleTail}>
			<StatusDot
				tone={liveTail && !tailPaused ? 'success' : 'neutral'}
				pulse={liveTail && !tailPaused}
			/>
			Live tail
		</Button>
		<Button size="sm" variant="outline" icon={icons.plus} onclick={() => (saveOpen = true)}>
			Save filters
		</Button>
	{/snippet}
</PageHeader>

<section class="flex flex-col gap-4">
	<Card class="flex flex-col gap-4">
		<Input
			type="search"
			bind:value={query}
			oninput={onSearchInput}
			placeholder="Search messages…"
			aria-label="Search messages"
		/>

		<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
			<label class="flex flex-col gap-1">
				<span class="text-fc-xs text-fc-fg-muted">App</span>
				<Select
					bind:value={selectedApp}
					onchange={apply}
					class="font-fc-mono"
					aria-label="Filter by app"
				>
					<option value="">All apps</option>
					{#each apps as app (app.name)}
						<option value={app.name}>{app.name} ({app.count})</option>
					{/each}
				</Select>
			</label>

			<div class="flex flex-col gap-1">
				<span class="text-fc-xs text-fc-fg-muted">Level</span>
				<div class="flex h-11 flex-wrap items-center gap-1.5">
					{#each LEVELS as level (level)}
						{@const on = selectedLevels.includes(level)}
						<button
							type="button"
							aria-pressed={on}
							class="rounded-fc-pill px-2.5 py-1 text-fc-xs uppercase focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fc-ring {on
								? 'bg-fc-accent text-fc-accent-fg'
								: levelChipClass(level)}"
							onclick={() => toggleLevel(level)}
						>
							{level}
						</button>
					{/each}
				</div>
			</div>

			<div class="flex flex-col gap-1">
				<span class="text-fc-xs text-fc-fg-muted">Source</span>
				<Select
					bind:value={source}
					onchange={apply}
					aria-label="Filter by source"
				>
					<option value="">All sources</option>
					<option value="browser">Client (browser)</option>
					<option value="server">Server (no source)</option>
				</Select>
			</div>

			<label class="flex flex-col gap-1">
				<span class="text-fc-xs text-fc-fg-muted">Saved query</span>
				<Select
					value=""
					onchange={(event) => applySaved(event.currentTarget.value)}
					aria-label="Apply a saved query"
				>
					<option value="">Apply a saved query…</option>
					{#each savedQueries as saved (saved.id)}
						<option value={String(saved.id)}>{saved.name}</option>
					{/each}
				</Select>
			</label>
		</div>

		<div class="flex flex-col gap-4">
			<Tabs
				items={RANGES.map((entry) => ({ id: entry.id, label: entry.label }))}
				value={range}
				label="Time range"
				onChange={(id) => {
					range = id;
					apply();
				}}
			/>
			{#if range === 'custom'}
				<div class="grid gap-4 sm:grid-cols-2">
					<label class="flex flex-col gap-1">
						<span class="text-fc-xs text-fc-fg-muted">From</span>
						<Input type="datetime-local" bind:value={customSince} onchange={apply} />
					</label>
					<label class="flex flex-col gap-1">
						<span class="text-fc-xs text-fc-fg-muted">To</span>
						<Input type="datetime-local" bind:value={customUntil} onchange={apply} />
					</label>
				</div>
			{/if}
		</div>

		{#if activeCount > 0}
			<div class="flex flex-wrap items-center gap-2 border-t border-fc-border pt-4">
				{#if requestId}
					<button
						type="button"
						class="flex max-w-full items-center gap-1 rounded-fc-pill bg-fc-surface px-2.5 py-1 font-fc-mono text-fc-xs"
						onclick={() => {
							requestId = '';
							apply();
						}}
					>
						<span class="truncate">req:{requestId}</span>
						<iconify-icon icon={icons.close} width="12" height="12" class="block"></iconify-icon>
					</button>
				{/if}
				{#if sessionId}
					<button
						type="button"
						title={sessionId}
						class="flex max-w-[16rem] items-center gap-1 rounded-fc-pill bg-fc-surface px-2.5 py-1 font-fc-mono text-fc-xs"
						onclick={() => {
							sessionId = '';
							apply();
						}}
					>
						<span class="truncate">session:{sessionId}</span>
						<iconify-icon icon={icons.close} width="12" height="12" class="block"></iconify-icon>
					</button>
				{/if}
				<Button size="sm" variant="ghost" icon={icons.close} onclick={clearFilters}>
					Clear {activeCount} filter{activeCount === 1 ? '' : 's'}
				</Button>
			</div>
		{/if}
	</Card>

	<Card>
		<LogHistogram {hist} onSelect={zoomTo} />
	</Card>
</section>

<section class="flex flex-col gap-4">
	{#if error}
		<Alert tone="danger" title="Could not load logs">{error}</Alert>
	{/if}

	<LogTable
		{entries}
		{gapIds}
		{loading}
		emptyDescription="Widen the time range, drop a level filter, or check that the app is shipping to this instance."
		onPivotApp={pivotApp}
		onPivotLevel={toggleLevel}
		onPivotRequest={pivotRequest}
		onPivotSession={pivotSession}
		onPivotSource={pivotSource}
		onContext={(id) => (contextAnchor = id)}
	/>

	{#if nextBefore && entries.length > 0}
		<div class="flex justify-center">
			<Button variant="outline" disabled={loadingMore} onclick={loadMore}>
				{loadingMore ? 'Loading…' : 'Load more'}
			</Button>
		</div>
	{/if}
</section>

<LogContextDrawer bind:anchorId={contextAnchor} />

<Modal bind:open={saveOpen} title="Save these filters" size="sm">
	<label class="flex flex-col gap-1">
		<span class="text-fc-sm text-fc-fg-muted">Name</span>
		<Input bind:value={saveName} placeholder="Nuage errors" />
	</label>
	{#snippet footer()}
		<div class="flex justify-end gap-2">
			<Button variant="ghost" onclick={() => (saveOpen = false)}>Cancel</Button>
			<Button disabled={saving || !saveName.trim()} onclick={saveQuery}>
				{saving ? 'Saving…' : 'Save'}
			</Button>
		</div>
	{/snippet}
</Modal>
