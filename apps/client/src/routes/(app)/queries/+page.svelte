<script lang="ts">
	import { onMount } from 'svelte';
	import { Alert, Button, ConfirmModal, EmptyState, Spinner, Table, icons, toast } from '@facile/muse';
	import { backend, type SavedQuery } from '$lib/backend';
	import { formatDate } from '$lib/format';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import { useT } from '$lib/i18n';

	const t = useT();

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
		return parts.length ? parts.join(' · ') : t('queries.noFilters');
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
			error = err instanceof Error ? err.message : t('queries.loadFailed');
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
			toast.success(t('queries.deleted', { name: saved.name }));
		} catch (err) {
			toast.danger(err instanceof Error ? err.message : t('queries.deleteFailed'));
		} finally {
			pending = null;
		}
	}

	onMount(load);
</script>

<svelte:head><title>{t('queries.title')} — Journal</title></svelte:head>

<PageHeader
	title={t('queries.title')}
	description={t('queries.description')}
>
	{#snippet actions()}
		<Button size="sm" variant="outline" href="/logs" icon={icons.plus}>{t('queries.saveNew')}</Button>
	{/snippet}
</PageHeader>

{#if error}
	<Alert tone="danger" title={t('queries.loadFailed')}>{error}</Alert>
{:else if loading}
	<div class="flex justify-center py-16"><Spinner /></div>
{:else if queries.length === 0}
	<EmptyState
		icon={icons.filter}
		title={t('queries.empty')}
		description={t('queries.emptyHint')}
	>
		<Button href="/logs" icon={icons.history}>{t('queries.openExplorer')}</Button>
	</EmptyState>
{:else}
	<Table>
		<thead>
			<tr>
				<th scope="col">{t('common.name')}</th>
				<th scope="col">{t('queries.filters')}</th>
				<th scope="col">{t('common.created')}</th>
				<th scope="col" aria-label={t('common.actions')}></th>
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
							{t('common.delete')}
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
	title={t('queries.deleteTitle')}
	description={t('queries.deleteBody')}
	confirmLabel={t('common.delete')}
	onConfirm={remove}
	onCancel={() => (pending = null)}
/>
