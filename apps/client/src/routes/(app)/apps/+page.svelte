<script lang="ts">
	import { onMount } from 'svelte';
	import { Alert, Button, EmptyState, Spinner, Table, icons } from '@facile/muse';
	import { backend, type AppSummary } from '$lib/backend';
	import { formatCount, formatDate, formatRelative } from '$lib/format';
	import PageHeader from '$lib/components/PageHeader.svelte';

	let apps = $state<AppSummary[]>([]);
	let loading = $state(true);
	let error = $state('');

	const sorted = $derived([...apps].sort((a, b) => b.count - a.count));

	onMount(async () => {
		try {
			apps = (await backend.listApps()).apps;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load apps';
		} finally {
			loading = false;
		}
	});
</script>

<svelte:head><title>Apps — Journal</title></svelte:head>

<PageHeader title="Apps" description="Every source that has ever shipped an entry to this instance." />

{#if error}
	<Alert tone="danger" title="Could not load apps">{error}</Alert>
{:else if loading}
	<div class="flex justify-center py-16"><Spinner /></div>
{:else if sorted.length === 0}
	<EmptyState
		icon={icons.server}
		title="No app has shipped a log yet"
		description="Create an ingest key, point the app's JOURNAL_URL at this instance, and its first entry will name it here."
	>
		<Button href="/settings/api" icon={icons.key}>Create an ingest key</Button>
	</EmptyState>
{:else}
	<Table>
		<thead>
			<tr>
				<th scope="col">App</th>
				<th scope="col">Entries</th>
				<th scope="col">Last seen</th>
				<!-- `aria-label` rather than an `sr-only` span: `sr-only` is absolutely positioned
				     and escapes the Table's horizontal scroller onto the document. -->
				<th scope="col" aria-label="Actions"></th>
			</tr>
		</thead>
		<tbody>
			{#each sorted as app (app.name)}
				<tr>
					<td class="font-fc-mono text-fc-xs">{app.name}</td>
					<td class="tabular-nums">{formatCount(app.count)}</td>
					<td class="whitespace-nowrap text-fc-fg-muted" title={formatDate(app.last_seen)}>
						{formatRelative(app.last_seen)}
					</td>
					<td class="text-right">
						<Button
							size="sm"
							variant="ghost"
							href="/logs?app={encodeURIComponent(app.name)}"
							iconRight={icons.arrow}
						>
							Logs
						</Button>
					</td>
				</tr>
			{/each}
		</tbody>
	</Table>
{/if}
