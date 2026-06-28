<script lang="ts">
	import { onMount } from 'svelte';
	import { backend, type AppSummary, type LogEntry, type LogLevel, type ListLogsParams } from '$lib/backend';

	const levels: LogLevel[] = ['debug', 'info', 'warn', 'error'];

	let apps = $state<AppSummary[]>([]);
	let entries = $state<LogEntry[]>([]);
	let selectedApp = $state<string | null>(null);
	let selectedLevels = $state<LogLevel[]>([]);
	let query = $state('');
	let nextBefore = $state<number | null>(null);
	let loading = $state(false);
	let loadingMore = $state(false);
	let liveTail = $state(false);
	let expandedId = $state<number | null>(null);
	let error = $state('');

	let searchTimer: ReturnType<typeof setTimeout> | undefined;

	function filterParams(): ListLogsParams {
		return {
			app: selectedApp ?? undefined,
			level: selectedLevels.length ? selectedLevels : undefined,
			q: query.trim() || undefined
		};
	}

	async function load() {
		loading = true;
		error = '';
		try {
			const res = await backend.listLogs({ ...filterParams(), limit: 100 });
			entries = res.entries;
			nextBefore = res.next_before;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load logs';
		} finally {
			loading = false;
		}
	}

	async function loadMore() {
		if (nextBefore == null || loadingMore) return;
		loadingMore = true;
		try {
			const res = await backend.listLogs({ ...filterParams(), limit: 100, before: nextBefore });
			entries = [...entries, ...res.entries];
			nextBefore = res.next_before;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load more';
		} finally {
			loadingMore = false;
		}
	}

	async function poll() {
		try {
			const res = await backend.listLogs({ ...filterParams(), limit: 100 });
			const maxId = entries.reduce((max, entry) => Math.max(max, entry.id), 0);
			const fresh = res.entries.filter((entry) => entry.id > maxId);
			if (fresh.length) entries = [...fresh, ...entries];
		} catch {
			/* keep tailing silently */
		}
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
		load();
	}

	function toggleLevel(level: LogLevel) {
		selectedLevels = selectedLevels.includes(level)
			? selectedLevels.filter((value) => value !== level)
			: [...selectedLevels, level];
		load();
	}

	function onSearchInput() {
		clearTimeout(searchTimer);
		searchTimer = setTimeout(load, 300);
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
			<button
				class="inline-flex h-9 items-center gap-2 rounded-md border border-border px-3 text-sm font-medium transition-colors {liveTail ? 'bg-primary text-primary-foreground' : 'bg-background hover:bg-accent'}"
				onclick={() => (liveTail = !liveTail)}
			>
				<span class="inline-block h-2 w-2 rounded-full {liveTail ? 'animate-pulse bg-primary-foreground' : 'bg-muted-foreground'}"></span>
				Live tail
			</button>
		</header>

		{#if error}
			<div class="border-b border-border bg-destructive/10 px-5 py-2 text-sm text-destructive">{error}</div>
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
								<tr class="border-b border-border/60 bg-muted/40">
									<td colspan="4" class="px-5 py-3">
										<div class="mb-2 font-mono text-xs whitespace-pre-wrap break-words">{entry.message}</div>
										{#if entry.meta && Object.keys(entry.meta).length > 0}
											<pre class="overflow-x-auto rounded-md border border-border bg-card p-3 font-mono text-xs">{JSON.stringify(entry.meta, null, 2)}</pre>
										{:else}
											<p class="text-xs text-muted-foreground">No metadata.</p>
										{/if}
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
