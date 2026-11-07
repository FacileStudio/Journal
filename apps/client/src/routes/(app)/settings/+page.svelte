<script lang="ts">
	import { getContext } from 'svelte';
	import { ProfileSection } from '@facile/muse';
	import type { AuthUser } from '$lib/backend';
	import { formatDate } from '$lib/format';
	import { useT } from '$lib/i18n';

	const t = useT();

	const auth = getContext<{ user: AuthUser | null }>('auth');
	const user = $derived(auth?.user ?? null);
</script>

<ProfileSection
	name={user?.name || user?.email || '—'}
	email={user?.email}
	avatar={user?.avatar_url || undefined}
	role={user?.is_admin ? t('profile.admin') : t('profile.member')}
>
	{#if user}
		<p class="text-fc-sm text-fc-fg-muted">{t('profile.since')} {formatDate(user.created_at)}</p>
	{/if}
</ProfileSection>
