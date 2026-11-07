<script lang="ts">
	import { onMount } from 'svelte';
	import { SecretField, SettingsRow, SettingsSection, StatusDot } from '@facile/muse';
	import { backend } from '$lib/backend';
	import { useT } from '$lib/i18n';

	const t = useT();

	let reachable = $state<boolean | null>(null);

	const apiBase = $derived(
		typeof window === 'undefined' ? backend.baseUrl : `${window.location.origin}${backend.baseUrl}`
	);

	onMount(async () => {
		try {
			await backend.authConfig();
			reachable = true;
		} catch {
			reachable = false;
		}
	});
</script>

<div class="flex flex-col gap-10">
	<SettingsSection title={t('advanced.instance')} description={t('advanced.instanceHint')}>
		<SettingsRow label={t('advanced.apiBase')} stacked>
			<SecretField value={apiBase} sensitive={false} />
		</SettingsRow>
		<SettingsRow label={t('common.status')} description={t('advanced.statusHint')}>
			{#if reachable === null}
				<StatusDot tone="neutral" label={t('advanced.checking')} pulse />
			{:else if reachable}
				<StatusDot tone="success" label={t('advanced.reachable')} />
			{:else}
				<StatusDot tone="danger" label={t('advanced.unreachable')} />
			{/if}
		</SettingsRow>
		<SettingsRow label={t('advanced.signIn')} description={t('advanced.signInHint')}>
			<span class="text-fc-sm text-fc-fg">{t('advanced.signInValue')}</span>
		</SettingsRow>
	</SettingsSection>

	<SettingsSection title={t('advanced.retention')} description={t('advanced.retentionHint')}>
		<SettingsRow label={t('advanced.where')} description={t('advanced.whereHint')}>
			<span class="font-fc-mono text-fc-sm text-fc-fg-muted">RETENTION_DAYS</span>
		</SettingsRow>
	</SettingsSection>
</div>
