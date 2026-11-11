<script lang="ts">
	import { Badge, Button, Card, icons } from '@facile/muse';
	import type { PendingAction } from '$lib/backend';
	import { formatDate } from '$lib/format';
	import { useT } from '$lib/i18n';

	let {
		action,
		busy,
		onconfirm,
		ondismiss
	}: {
		action: PendingAction;
		busy: boolean;
		onconfirm: (action: PendingAction) => void;
		ondismiss: (action: PendingAction) => void;
	} = $props();

	const t = useT();

	const filters = $derived.by(() => {
		const query = action.query;
		if (!query) return '';
		const parts: string[] = [];
		if (query.app) parts.push(`app:${query.app}`);
		if (query.levels?.length) parts.push(`levels:${query.levels.join(',')}`);
		if (query.q) parts.push(`q:${query.q}`);
		return parts.join(' · ');
	});

	/** Names the client that asked: Bulle, the host of a client URL, else its id. */
	function clientName(clientId: string): string {
		if (clientId === 'bulle') return 'Bulle';
		try {
			return new URL(clientId).host;
		} catch {
			return clientId || '?';
		}
	}
</script>

<Card class="flex flex-col gap-4">
	<div class="flex flex-wrap items-center justify-between gap-2">
		<Badge tone="danger">
			{action.kind === 'delete_alert' ? t('pending.deleteAlert') : t('pending.deleteQuery')}
		</Badge>
		<span class="text-fc-xs text-fc-fg-muted">
			{t('pending.askedBy', { client: clientName(action.client_id) })} · {formatDate(action.created_at)}
		</span>
	</div>

	{#if action.query}
		<div class="flex flex-col gap-1 rounded-fc-md bg-fc-surface p-4">
			<p class="text-fc-md font-medium break-words text-fc-fg">{action.query.name}</p>
			{#if filters}
				<p class="font-fc-mono text-fc-xs break-words text-fc-fg-muted">{filters}</p>
			{/if}
			<p class="text-fc-sm text-fc-fg-muted">{t('pending.sharedQuery')}</p>
		</div>
	{:else if action.alert}
		<div class="flex flex-col gap-1 rounded-fc-md bg-fc-surface p-4">
			<div class="flex flex-wrap items-center gap-2">
				<p class="text-fc-md font-medium break-words text-fc-fg">{action.alert.name}</p>
				{#if !action.alert.enabled}
					<Badge tone="neutral">{t('pending.alertOff')}</Badge>
				{/if}
			</div>
			<p class="text-fc-sm break-words text-fc-fg-muted">
				{t('pending.alertRule', {
					threshold: String(action.alert.threshold),
					minutes: String(action.alert.window_minutes),
					query: action.alert.saved_query
				})}
			</p>
		</div>
	{:else}
		<p class="text-fc-sm break-words text-fc-fg-muted">{action.summary}</p>
	{/if}

	{#if action.state === 'changed'}
		<p class="text-fc-sm text-fc-warning">{t('pending.changed')}</p>
	{:else if action.state === 'missing'}
		<p class="text-fc-sm text-fc-warning">{t('pending.missing')}</p>
	{/if}

	<div class="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
		<Button variant="ghost" class="w-full sm:w-auto" disabled={busy} onclick={() => ondismiss(action)}>
			{t('pending.dismiss')}
		</Button>
		{#if action.state === 'ready'}
			<Button
				variant="danger"
				class="w-full sm:w-auto"
				disabled={busy}
				icon={icons.remove}
				onclick={() => onconfirm(action)}
			>
				{t('pending.confirmDelete')}
			</Button>
		{/if}
	</div>
</Card>
