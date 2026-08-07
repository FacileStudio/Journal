<script lang="ts">
	import { untrack } from 'svelte';
	import { Alert, Button, Drawer, Spinner, icons } from '@facile/muse';
	import { backend, type LogEntry } from '$lib/backend';
	import { formatTime } from '$lib/format';
	import LevelBadge from './LevelBadge.svelte';

	const MAX = 200;
	const STEP = 50;

	let { anchorId = $bindable<number | null>(null) }: { anchorId?: number | null } = $props();

	let entries = $state<LogEntry[]>([]);
	let resolvedAnchor = $state<number | null>(null);
	let before = $state(STEP);
	let after = $state(STEP);
	let loading = $state(false);
	let error = $state('');

	const open = $derived(anchorId !== null);

	/*
	 * `inFlight` is the anchor the current request belongs to. Widening the window refetches the
	 * whole range rather than appending, so an answer for a previous anchor must never land.
	 */
	let inFlight = 0;

	async function fetchContext(id: number) {
		inFlight = id;
		loading = true;
		error = '';
		try {
			const res = await backend.logContext(id, before, after);
			if (inFlight !== id) return;
			entries = res.entries;
			resolvedAnchor = res.anchor_id;
		} catch (err) {
			if (inFlight !== id) return;
			error = err instanceof Error ? err.message : 'Failed to load context';
		} finally {
			if (inFlight === id) loading = false;
		}
	}

	/*
	 * `untrack` is load-bearing. `fetchContext` reads `before`/`after`, so without it the effect
	 * depends on the two values it also resets — widening the window would fire the effect, snap
	 * the window back to 50 and refetch, silently undoing every "Load older". Only a new anchor
	 * may re-arm this.
	 */
	$effect(() => {
		const id = anchorId;
		if (id === null) return;
		untrack(() => {
			before = STEP;
			after = STEP;
			entries = [];
			resolvedAnchor = null;
			void fetchContext(id);
		});
	});

	function extend(side: 'before' | 'after') {
		const id = anchorId;
		if (id === null) return;
		if (side === 'before') before = Math.min(before + STEP, MAX);
		else after = Math.min(after + STEP, MAX);
		void fetchContext(id);
	}
</script>

<Drawer
	{open}
	title="Log context"
	description="The unfiltered stream around this entry."
	showClose
	onClose={() => (anchorId = null)}
	class="max-w-fc-md"
>
	<div class="flex flex-col gap-3">
		{#if error}
			<Alert tone="danger" title="Could not load context">{error}</Alert>
		{/if}

		<Button
			size="sm"
			variant="ghost"
			icon={icons.chevronUp}
			disabled={loading || after >= MAX}
			onclick={() => extend('after')}
		>
			{after >= MAX ? 'Newer limit reached' : 'Load newer'}
		</Button>

		{#if loading && entries.length === 0}
			<div class="flex justify-center py-10"><Spinner /></div>
		{:else if entries.length === 0}
			<p class="py-10 text-center text-fc-sm text-fc-fg-muted">No surrounding entries.</p>
		{:else}
			<ul class="flex flex-col">
				{#each entries as entry (entry.id)}
					<li
						class="flex flex-col gap-1 border-t border-fc-border py-2 first:border-t-0 {entry.id ===
						resolvedAnchor
							? 'bg-fc-surface'
							: ''}"
					>
						<div class="flex flex-wrap items-center gap-2">
							<span class="font-fc-mono text-fc-xs text-fc-fg-muted">
								{formatTime(entry.created_at)}
							</span>
							<span class="rounded-fc-pill bg-fc-surface px-2 py-0.5 font-fc-mono text-fc-xs">
								{entry.app}
							</span>
							<LevelBadge level={entry.level} />
						</div>
						<p class="font-fc-mono text-fc-xs break-words whitespace-pre-wrap text-fc-fg">
							{entry.message}
						</p>
					</li>
				{/each}
			</ul>
		{/if}

		<Button
			size="sm"
			variant="ghost"
			icon={icons.chevronDown}
			disabled={loading || before >= MAX}
			onclick={() => extend('before')}
		>
			{before >= MAX ? 'Older limit reached' : 'Load older'}
		</Button>
	</div>
</Drawer>
