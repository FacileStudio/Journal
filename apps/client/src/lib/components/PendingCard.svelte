<script lang="ts">
	import { Badge, PendingCard as Shell, icons } from '@facile/muse';
	import type { PendingAction } from '$lib/backend';
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
</script>

<Shell
	kind={action.kind === 'delete_alert' ? t('pending.deleteAlert') : t('pending.deleteQuery')}
	client={action.client_id}
	createdAt={action.created_at}
	state={action.state}
	{busy}
	confirmLabel={t('pending.confirmDelete')}
	confirmIcon={icons.remove}
	changedLabel={t('pending.changed')}
	missingLabel={t('pending.missing')}
	onConfirm={() => onconfirm(action)}
	onDismiss={() => ondismiss(action)}
>
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
</Shell>
