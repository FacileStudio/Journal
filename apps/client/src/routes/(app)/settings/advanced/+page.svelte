<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { Button, ConfirmModal, DangerZone, SecretField, SettingsRow, SettingsSection, StatusDot, icons } from '@facile/muse';
	import { backend } from '$lib/backend';
	import { useT } from '$lib/i18n';

	const t = useT();

	let reachable = $state<boolean | null>(null);
	let confirmOpen = $state(false);
	let deleting = $state(false);
	let deleteError = $state<string | null>(null);

	async function deleteAccount() {
		deleting = true;
		deleteError = null;
		try {
			await backend.deleteMe();
			await goto('/login');
		} catch (error) {
			deleteError = error instanceof Error ? error.message : t('profile.deleteFailed');
		} finally {
			deleting = false;
		}
	}

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
			<span class="font-fc-mono text-fc-sm text-fc-fg-muted">RETENTION_DAYS · RETENTION_MAX_ENTRIES</span>
		</SettingsRow>
	</SettingsSection>

	<DangerZone title={t('profile.danger')} description={t('profile.dangerHint')}>
		<div class="flex flex-col gap-1">
			<p class="text-fc-sm font-medium text-fc-fg">{t('profile.delete')}</p>
			<p class="text-fc-sm text-fc-fg-muted">{t('profile.deleteHint')}</p>
		</div>
		<Button variant="danger" onclick={() => (confirmOpen = true)}>{t('profile.deleteButton')}</Button>
		{#if deleteError}
			<p class="w-full text-fc-sm text-fc-danger" role="alert">{deleteError}</p>
		{/if}
	</DangerZone>
</div>

<ConfirmModal
	bind:open={confirmOpen}
	tone="danger"
	icon={icons.logout}
	title={t('profile.confirmTitle')}
	description={t('profile.confirmBody')}
	confirmLabel={deleting ? t('profile.deleting') : t('profile.confirmButton')}
	onConfirm={deleteAccount}
	onCancel={() => (confirmOpen = false)}
/>
