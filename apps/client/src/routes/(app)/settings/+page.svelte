<script lang="ts">
	import { getContext } from 'svelte';
	import { Badge, Button, SettingsRow, SettingsSection, icons } from '@facile/muse';
	import type { AuthUser } from '$lib/backend';
	import { formatDate } from '$lib/format';

	const auth = getContext<{ user: AuthUser | null; logout: () => void }>('auth');
	const user = $derived(auth?.user ?? null);
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
</div>
