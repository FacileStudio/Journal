<script lang="ts">
	import { getContext } from 'svelte';
	import { goto } from '$app/navigation';
	import { Badge, Button, ConfirmModal, SettingsRow, SettingsSection, icons } from '@facile/muse';
	import { backend, type AuthUser } from '$lib/backend';
	import { clearToken } from '$lib/auth';
	import { formatDate } from '$lib/format';

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
			deleteError = error instanceof Error ? error.message : 'Could not delete the account';
		} finally {
			deleting = false;
		}
	}
</script>

<div class="flex flex-col gap-10">
	<SettingsSection title="Account" description="Your dashboard identity on this instance.">
		<SettingsRow label="Name">
			<span class="text-fc-sm text-fc-fg">{user?.name || '—'}</span>
		</SettingsRow>
		<SettingsRow label="Email" description="Sign-in address. Changing it is not supported yet.">
			<span class="font-fc-mono text-fc-sm text-fc-fg">{user?.email ?? '—'}</span>
		</SettingsRow>
		<SettingsRow label="Role" description="Admins manage ingest keys and alert rules.">
			<Badge tone={user?.is_admin ? 'admin' : 'neutral'}>
				{user?.is_admin ? 'Admin' : 'Member'}
			</Badge>
		</SettingsRow>
		<SettingsRow label="Member since">
			<span class="text-fc-sm text-fc-fg-muted">
				{user ? formatDate(user.created_at) : '—'}
			</span>
		</SettingsRow>
	</SettingsSection>

	<SettingsSection title="Session" description="This browser holds a 30-day session token.">
		<SettingsRow label="Log out" description="Revokes the token on the server, not just here.">
			<Button variant="outline" icon={icons.logout} onclick={() => auth?.logout()}>Log out</Button>
		</SettingsRow>
	</SettingsSection>

	<SettingsSection
		title="Danger zone"
		description="Erasure under GDPR Article 17. The account, every session and the cached avatar are destroyed; log entries stay, because they name apps and never people."
	>
		<SettingsRow
			label="Delete this account"
			description="Permanent. The last administrator cannot delete the account until another admin exists."
		>
			<Button variant="danger" onclick={() => (confirmOpen = true)}>Delete account…</Button>
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
	title="Delete your account?"
	description="You are signed out everywhere immediately, and there is no way back. Log entries already stored remain — they carry no trace of you."
	confirmLabel={deleting ? 'Deleting…' : 'Delete forever'}
	onConfirm={deleteAccount}
	onCancel={() => (confirmOpen = false)}
/>
