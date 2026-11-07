<script lang="ts">
	import { getContext } from 'svelte';
	import { goto } from '$app/navigation';
	import { Avatar, Badge, Button, ConfirmModal, SettingsRow, SettingsSection, icons } from '@facile/muse';
	import { backend, type AuthUser } from '$lib/backend';
	import { clearToken } from '$lib/auth';
	import { formatDate } from '$lib/format';
	import { useT } from '$lib/i18n';

	const t = useT();

	const auth = getContext<{ user: AuthUser | null; logout: () => void }>('auth');
	const user = $derived(auth?.user ?? null);

	let confirmOpen = $state(false);
	let deleting = $state(false);
	let deleteError = $state<string | null>(null);

	async function deleteAccount() {
		deleting = true;
		deleteError = null;
		try {
			await backend.deleteMe();
			clearToken();
			await goto('/login');
		} catch (error) {
			deleteError = error instanceof Error ? error.message : t('profile.deleteFailed');
		} finally {
			deleting = false;
		}
	}
</script>

<div class="flex flex-col gap-10">
	<SettingsSection title={t('profile.account')} description={t('profile.accountHint')}>
		<SettingsRow label={t('common.name')}>
			<span class="flex items-center gap-2.5">
				<Avatar src={user?.avatar_url || undefined} name={user?.name || user?.email || ''} size="sm" />
				<span class="text-fc-sm text-fc-fg">{user?.name || '—'}</span>
			</span>
		</SettingsRow>
		<SettingsRow label={t('profile.email')} description={t('profile.emailHint')}>
			<span class="font-fc-mono text-fc-sm text-fc-fg">{user?.email ?? '—'}</span>
		</SettingsRow>
		<SettingsRow label={t('profile.role')} description={t('profile.roleHint')}>
			<Badge tone={user?.is_admin ? 'admin' : 'neutral'}>
				{user?.is_admin ? t('profile.admin') : t('profile.member')}
			</Badge>
		</SettingsRow>
		<SettingsRow label={t('profile.since')}>
			<span class="text-fc-sm text-fc-fg-muted">
				{user ? formatDate(user.created_at) : '—'}
			</span>
		</SettingsRow>
	</SettingsSection>

	<SettingsSection title={t('profile.session')} description={t('profile.sessionHint')}>
		<SettingsRow label={t('profile.logout')} description={t('profile.logoutHint')}>
			<Button variant="outline" icon={icons.logout} onclick={() => auth?.logout()}>{t('profile.logout')}</Button>
		</SettingsRow>
	</SettingsSection>

	<SettingsSection title={t('profile.danger')} description={t('profile.dangerHint')}>
		<SettingsRow label={t('profile.delete')} description={t('profile.deleteHint')}>
			<Button variant="danger" onclick={() => (confirmOpen = true)}>{t('profile.deleteButton')}</Button>
		</SettingsRow>
		{#if deleteError}
			<p class="text-fc-sm text-fc-danger" role="alert">{deleteError}</p>
		{/if}
	</SettingsSection>
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
