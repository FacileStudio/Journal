<script lang="ts">
	import { Icon, Button, EmptyState, Spinner, Table, icons, toast } from '@facile/muse';
	import { backend, type LogEntry, type LogLevel, type ResolvedStack, type StackFrame } from '$lib/backend';
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
		onPivotSession,
		onPivotSource,
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
		onPivotSession?: (sessionId: string) => void;
		onPivotSource?: (source: string) => void;
		onContext?: (id: number) => void;
	} = $props();

	let expandedId = $state<number | null>(null);

	const gaps = $derived(new Set(gapIds));

	function toggle(id: number) {
		if (expandedId !== id) void resolveStack(id);
		expandedId = expandedId === id ? null : id;
	}

	/* Only browser entries carry a source, so this is what turns an expanded
	   row into the "show me every client-side error" view. */
	function sourceOf(entry: LogEntry): string | null {
		const value = entry.meta?.['source'];
		return typeof value === 'string' && value ? value : null;
	}

	function sourceLabel(source: string | null): string {
		switch (source) {
			case 'browser': return 'browser';
			case 'collector': return 'collector';
			default: return 'server';
		}
	}

	function requestIdOf(entry: LogEntry): string | null {
		const value = entry.meta?.['request_id'];
		if (typeof value === 'string' && value) return value;
		if (typeof value === 'number') return String(value);
		return null;
	}

	/* Only the browser endpoint writes this one. It is what turns a single error
	   into everything else that tab did. */
	function sessionIdOf(entry: LogEntry): string | null {
		const value = entry.meta?.['session_id'];
		return typeof value === 'string' && value ? value : null;
	}

	/* A UUID in a pivot button is mostly noise: the first segment is enough to
	   recognise, and the whole value still goes to the filter. */
	function shortId(value: string): string {
		return value.length > 12 ? `${value.slice(0, 8)}…` : value;
	}

	/* A stack trace inside JSON.stringify is one long line of \n escapes, which
	   is the difference between a usable browser error and an unreadable one.
	   It comes out into its own block and the rest of meta keeps the JSON. */
	/* One resolved stack per expanded row, fetched when the row opens rather
	   than with the page: resolving is the expensive half of a source map, a
	   page holds a hundred entries, and a reader opens one. */
	let resolved = $state<Record<number, ResolvedStack | 'loading'>>({});

	async function resolveStack(id: number) {
		if (resolved[id]) return;
		resolved[id] = 'loading';
		try {
			resolved[id] = await backend.logStack(id);
		} catch {
			/* The raw stack is already on screen; failing to improve it is not
			   worth an error message. */
			delete resolved[id];
		}
	}

	function frameLabel(frame: StackFrame): string {
		const where = frame.resolved
			? `${frame.source}:${frame.source_line}:${frame.source_column}`
			: `${frame.file ?? ''}${frame.line ? `:${frame.line}:${frame.column}` : ''}`;
		const fn = frame.name || frame.function;
		return fn ? `${fn} — ${where}` : where;
	}

	function stackOf(entry: LogEntry): string | null {
		const value = entry.meta?.['stack'];
		return typeof value === 'string' && value ? value : null;
	}

	function metaOf(entry: LogEntry): string | null {
		if (!entry.meta) return null;
		const rest = { ...entry.meta };
		delete rest['stack'];
		if (Object.keys(rest).length === 0) return null;
		return JSON.stringify(rest, null, 2);
	}

	/** Renders the whole entry as a plain-text block, ready to paste into another tool. */
	function contextText(e: LogEntry): string {
		const lines = [
			`ID:           ${e.id}`,
			`App:          ${e.app}`,
			`Level:        ${e.level}`,
			`Created:      ${e.created_at}`,
			`Received:     ${e.received_at}`,
			`Message:      ${e.message}`
		];
		if (e.meta && Object.keys(e.meta).length > 0) {
			lines.push('', 'Meta:');
			for (const [key, value] of Object.entries(e.meta)) {
				let rendered: string;
				try {
					rendered = JSON.stringify(value);
				} catch {
					rendered = String(value);
				}
				lines.push(`  ${key}: ${rendered}`);
			}
		}
		return lines.join('\n');
	}

	function copyEntry(entry: LogEntry) {
		const text = contextText(entry);
		if (navigator.clipboard && navigator.clipboard.writeText) {
			navigator.clipboard
				.writeText(text)
				.then(() => toast.success('Entry context copied to clipboard.'))
				.catch(() => toast.danger('Could not copy to clipboard.'));
		} else {
			toast.danger('Clipboard is not available in this browser.');
		}
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
				<th scope="col" class="hidden sm:table-cell">Source</th>
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
					<td class="hidden sm:table-cell">
						<button
							type="button"
							class="rounded-fc-pill bg-fc-surface px-2 py-0.5 font-fc-mono text-fc-xs text-fc-fg hover:bg-fc-accent hover:text-fc-accent-fg focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fc-ring"
							title="Filter by source: {sourceOf(entry) ?? 'server'}"
							onclick={(event) => {
								event.stopPropagation();
								onPivotSource?.(sourceOf(entry) ?? 'server');
							}}
						>
							{sourceLabel(sourceOf(entry))}
						</button>
					</td>
					<!-- The message is the reason anyone opened this page. This column claims
					     the remaining table width and the preview clamps at two lines with an
					     ellipsis, so a row shows the start of the body without expanding it. -->
					<td class="w-full">
						<button
							type="button"
							class="block w-full line-clamp-2 text-left font-fc-mono text-fc-xs text-fc-fg focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fc-ring"
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
						<td colspan="5">
							<div class="flex flex-col gap-4 py-2">
								<p class="font-fc-mono text-fc-xs break-words whitespace-pre-wrap text-fc-fg">
									{entry.message}
								</p>
								{#if stackOf(entry)}
									{@const stack = resolved[entry.id]}
									{#if stack && stack !== 'loading' && stack.resolved > 0}
										<!-- Resolved frames read as source locations; the ones no map
										     explained keep their bundle position so the trace stays whole. -->
										<div
											class="overflow-x-auto rounded-fc-sm bg-fc-bg p-3 font-fc-mono text-fc-xs leading-relaxed"
										>
											{#each stack.frames as frame (frame.raw)}
												<div class={frame.resolved ? 'text-fc-fg' : 'text-fc-fg-muted'}>
													{frameLabel(frame)}
												</div>
											{/each}
										</div>
										<p class="text-fc-xs text-fc-fg-muted">
											{stack.resolved} of {stack.frames.length} frames mapped from release
											<span class="font-fc-mono">{stack.release}</span>.
										</p>
									{:else}
										<pre
											class="overflow-x-auto rounded-fc-sm bg-fc-bg p-3 font-fc-mono text-fc-xs leading-relaxed text-fc-fg-muted">{stackOf(
												entry
											)}</pre>
										{#if stack && stack !== 'loading' && stack.release && stack.resolved === 0}
											<p class="text-fc-xs text-fc-fg-muted">
												No source map uploaded for release
												<span class="font-fc-mono">{stack.release}</span>.
											</p>
										{/if}
									{/if}
								{/if}
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
									<Button
										size="sm"
										variant="outline"
										icon={icons.copy}
										onclick={() => copyEntry(entry)}
									>
										Copy
									</Button>
									{#if sourceOf(entry) && onPivotSource}
										<Button
											size="sm"
											variant="ghost"
											icon={icons.filter}
											onclick={() => onPivotSource?.(sourceOf(entry) as string)}
										>
											source: {sourceOf(entry)}
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
									{#if sessionIdOf(entry) && onPivotSession}
										<Button
											size="sm"
											variant="ghost"
											icon={icons.filter}
											onclick={() => onPivotSession?.(sessionIdOf(entry) as string)}
										>
											session: {shortId(sessionIdOf(entry) as string)}
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
						<td colspan="5">
							<span class="flex items-center gap-2 text-fc-xs text-fc-warning">
								<Icon icon={icons.warning} size={14} class="block" />
								possible gap — some entries between these rows were never fetched
							</span>
						</td>
					</tr>
				{/if}
			{/each}
		</tbody>
	</Table>
{/if}
