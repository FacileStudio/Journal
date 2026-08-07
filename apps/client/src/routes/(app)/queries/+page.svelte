<script lang="ts">
	import { onMount } from 'svelte';
	import { Alert, Button, ConfirmModal, EmptyState, Spinner, Table, icons, toast } from '@facile/muse';
	import { backend, type SavedQuery } from '$lib/backend';
	import { formatDate } from '$lib/format';
	import PageHeader from '$lib/components/PageHeader.svelte';

	let queries = $state<SavedQuery[]>([]);
	let loading = $state(true);
	let error = $state('');
	let pending = $state<SavedQuery | null>(null);

	function summary(saved: SavedQuery): string {
		const parts: string[] = [];
		if (saved.params.app) parts.push(`app:${saved.params.app}`);
		if (saved.params.levels?.length) parts.push(`levels:${saved.params.levels.join(',')}`);
		if (saved.params.q) parts.push(`q:${saved.params.q}`);
		if (saved.params.request_id) parts.push(`req:${saved.params.request_id}`);
		return parts.length ? parts.join(' · ') : 'no filters';
	}

	function href(saved: SavedQuery): string {
		const params = new URLSearchParams();
		if (saved.params.app) params.set('app', saved.params.app);
		if (saved.params.levels?.length) params.set('level', saved.params.levels.join(','));
		if (saved.params.q) params.set('q', saved.params.q);
		if (saved.params.request_id) params.set('request_id', saved.params.request_id);
		return `/logs?${params}`;
	}

	async function load() {
		try {
			queries = (await backend.listQueries()).queries;
			error = '';
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load saved queries';
		} finally {
			loading = false;
		}
	}

	async function remove() {
		const saved = pending;
		if (!saved) return;
		try {
			await backend.deleteQuery(saved.id);
			await load();
			toast.success(`Deleted “${saved.name}”.`);
		} catch (err) {
			toast.danger(err instanceof Error ? err.message : 'Could not delete the query');
		} finally {
			pending = null;
		}
	}

	onMount(load);
</script>

<svelte:head><title>Saved queries — Journal</title></svelte:head>

<PageHeader
	title="Saved queries"
	description="Named filter sets. Alert rules are built on top of them."
>
	{#snippet actions()}
		<Button size="sm" variant="outline" href="/logs" icon={icons.plus}>Save a new one</Button>
	{/snippet}
</PageHeader>

{#if error}
	<Alert tone="danger" title="Could not load saved queries">{error}</Alert>
{:else if loading}
	<div class="flex justify-center py-16"><Spinner /></div>
{:else if queries.length === 0}
	<EmptyState
		icon={icons.filter}
		title="No saved queries yet"
		description="Set up a filter in the explorer, then save it — an alert rule can only watch a query that has a name."
	>
		<Button href="/logs" icon={icons.history}>Open the explorer</Button>
	</EmptyState>
{:else}
	<Table>
		<thead>
			<tr>
				<th scope="col">Name</th>
				<th scope="col">Filters</th>
				<th scope="col">Created</th>
				<th scope="col" aria-label="Actions"></th>
			</tr>
		</thead>
		<tbody>
			{#each queries as saved (saved.id)}
				<tr>
					<td class="font-medium">
						<a class="hover:underline" href={href(saved)}>{saved.name}</a>
					</td>
					<td class="font-fc-mono text-fc-xs text-fc-fg-muted">{summary(saved)}</td>
					<td class="whitespace-nowrap text-fc-fg-muted">{formatDate(saved.created_at)}</td>
					<td class="text-right">
						<Button
							size="sm"
							variant="ghost-danger"
							icon={icons.remove}
							onclick={() => (pending = saved)}
						>
							Delete
						</Button>
					</td>
				</tr>
			{/each}
		</tbody>
	</Table>
{/if}

<ConfirmModal
	open={pending !== null}
	tone="danger"
	icon={icons.remove}
	title="Delete this saved query?"
	description="Any alert rule built on it must be deleted first, and the filter set itself cannot be recovered."
	confirmLabel="Delete"
	onConfirm={remove}
	onCancel={() => (pending = null)}
/>
