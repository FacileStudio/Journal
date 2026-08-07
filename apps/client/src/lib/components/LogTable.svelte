<script lang="ts">
	import { Button, EmptyState, Spinner, Table, icons } from '@facile/muse';
	import type { LogEntry, LogLevel } from '$lib/backend';
	import { formatClock, formatTime } from '$lib/format';
	import LevelBadge from './LevelBadge.svelte';

	let {
		entries,
		gapIds = [],
		loading = false,
		emptyTitle = 'No log entries match these filters',
		emptyDescription,
		onPivotApp,
		onPivotLevel,
		onPivotRequest,
		onContext
	}: {
		entries: LogEntry[];
		gapIds?: number[];
		loading?: boolean;
		emptyTitle?: string;
		emptyDescription?: string;
		onPivotApp?: (app: string) => void;
		onPivotLevel?: (level: LogLevel) => void;
		onPivotRequest?: (requestId: string) => void;
		onContext?: (id: number) => void;
	} = $props();

	let expandedId = $state<number | null>(null);

	const gaps = $derived(new Set(gapIds));

	function toggle(id: number) {
		expandedId = expandedId === id ? null : id;
	}

	function requestIdOf(entry: LogEntry): string | null {
		const value = entry.meta?.['request_id'];
		if (typeof value === 'string' && value) return value;
		if (typeof value === 'number') return String(value);
		return null;
	}

	function metaOf(entry: LogEntry): string | null {
		if (!entry.meta || Object.keys(entry.meta).length === 0) return null;
		return JSON.stringify(entry.meta, null, 2);
	}
</script>

{#if loading && entries.length === 0}
	<div class="flex justify-center py-16"><Spinner /></div>
{:else if entries.length === 0}
	<EmptyState icon={icons.filter} title={emptyTitle} description={emptyDescription} />
{:else}
	<Table>
		<thead>
			<tr>
				<th scope="col" class="whitespace-nowrap">Time</th>
				<!-- The message is the reason anyone opened this page, so on a phone the columns
				     that can be recovered by expanding a row give up their width to it. -->
				<th scope="col" class="hidden sm:table-cell">App</th>
				<th scope="col">Level</th>
				<th scope="col" class="w-full">Message</th>
			</tr>
		</thead>
		<tbody>
			{#each entries as entry (entry.id)}
				<tr class="cursor-pointer hover:bg-fc-surface" onclick={() => toggle(entry.id)}>
					<td class="whitespace-nowrap font-fc-mono text-fc-xs text-fc-fg-muted">
						<span class="sm:hidden">{formatClock(entry.created_at)}</span>
						<span class="hidden sm:inline">{formatTime(entry.created_at)}</span>
					</td>
					<td class="hidden sm:table-cell">
						<button
							type="button"
							class="rounded-fc-pill bg-fc-surface px-2 py-0.5 font-fc-mono text-fc-xs text-fc-fg hover:bg-fc-accent hover:text-fc-accent-fg focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fc-ring"
							title="Filter by {entry.app}"
							onclick={(event) => {
								event.stopPropagation();
								onPivotApp?.(entry.app);
							}}
						>
							{entry.app}
						</button>
					</td>
					<td>
						<button
							type="button"
							class="rounded-fc-pill focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fc-ring"
							title="Filter by {entry.level}"
							onclick={(event) => {
								event.stopPropagation();
								onPivotLevel?.(entry.level);
							}}
						>
							<LevelBadge level={entry.level} />
						</button>
					</td>
					<!-- `max-w-0` is the only thing that makes a cell in an auto-layout table
					     truncate instead of growing to fit its longest line. -->
					<td class="max-w-0">
						<button
							type="button"
							class="block w-full truncate text-left font-fc-mono text-fc-xs text-fc-fg focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fc-ring"
							aria-expanded={expandedId === entry.id}
							onclick={(event) => {
								event.stopPropagation();
								toggle(entry.id);
							}}
						>
							{entry.message}
						</button>
					</td>
				</tr>

				{#if expandedId === entry.id}
					<tr class="bg-fc-surface">
						<td colspan="4">
							<div class="flex flex-col gap-4 py-2">
								<p class="font-fc-mono text-fc-xs break-words whitespace-pre-wrap text-fc-fg">
									{entry.message}
								</p>
								{#if metaOf(entry)}
									<pre
										class="overflow-x-auto rounded-fc-sm bg-fc-bg p-3 font-fc-mono text-fc-xs text-fc-fg">{metaOf(
											entry
										)}</pre>
								{:else}
									<p class="text-fc-xs text-fc-fg-muted">No metadata.</p>
								{/if}
								<div class="flex flex-wrap items-center gap-2">
									{#if onContext}
										<Button
											size="sm"
											variant="outline"
											icon={icons.clock}
											onclick={() => onContext?.(entry.id)}
										>
											Context
										</Button>
									{/if}
									{#if requestIdOf(entry) && onPivotRequest}
										<Button
											size="sm"
											variant="ghost"
											icon={icons.filter}
											onclick={() => onPivotRequest?.(requestIdOf(entry) as string)}
										>
											request_id: {requestIdOf(entry)}
										</Button>
									{/if}
									<span class="ml-auto text-fc-xs text-fc-fg-muted">
										<span class="font-fc-mono sm:hidden">{entry.app} · </span>received
										{formatTime(entry.received_at)}
									</span>
								</div>
							</div>
						</td>
					</tr>
				{/if}

				{#if gaps.has(entry.id)}
					<tr class="bg-fc-warning/10">
						<td colspan="4">
							<span class="flex items-center gap-2 text-fc-xs text-fc-warning">
								<iconify-icon icon={icons.warning} width="14" height="14" class="block"
								></iconify-icon>
								possible gap — some entries between these rows were never fetched
							</span>
						</td>
					</tr>
				{/if}
			{/each}
		</tbody>
	</Table>
{/if}
