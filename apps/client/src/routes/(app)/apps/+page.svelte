<script lang="ts">
	import { onMount } from 'svelte';
	import { Alert, Button, EmptyState, Spinner, Table, icons } from '@facile/muse';
	import { backend, type AppSummary } from '$lib/backend';
	import { formatCount, formatDate, formatRelative } from '$lib/format';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import { useT } from '$lib/i18n';

	const t = useT();

	let apps = $state<AppSummary[]>([]);
	let loading = $state(true);
	let error = $state('');

	const sorted = $derived([...apps].sort((a, b) => b.count - a.count));

	onMount(async () => {
		try {
			apps = (await backend.listApps()).apps;
		} catch (err) {
			error = err instanceof Error ? err.message : t('apps.loadFailed');
		} finally {
			loading = false;
		}
	});
</script>

<svelte:head><title>{t('apps.title')} — Journal</title></svelte:head>

<PageHeader title={t('apps.title')} description={t('apps.description')} />

{#if error}
	<Alert tone="danger" title={t('apps.loadFailed')}>{error}</Alert>
{:else if loading}
	<div class="flex justify-center py-16"><Spinner /></div>
{:else if sorted.length === 0}
	<EmptyState
		icon={icons.server}
		title={t('overview.noApp')}
		description={t('apps.emptyHint')}
	>
		<Button href="/settings/api" icon={icons.key}>{t('apps.createKey')}</Button>
	</EmptyState>
{:else}
	<Table>
		<thead>
			<tr>
				<th scope="col">{t('common.app')}</th>
				<th scope="col">{t('common.entries')}</th>
				<th scope="col">{t('common.lastSeen')}</th>
				<!-- `aria-label` rather than an `sr-only` span: `sr-only` is absolutely positioned
				     and escapes the Table's horizontal scroller onto the document. -->
				<th scope="col" aria-label={t('common.actions')}></th>
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
							{t('apps.logsLink')}
						</Button>
					</td>
				</tr>
			{/each}
		</tbody>
	</Table>
{/if}
