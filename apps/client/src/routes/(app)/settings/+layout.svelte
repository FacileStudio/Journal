<script lang="ts">
	import { getContext } from 'svelte';
	import { page } from '$app/state';
	import { SettingsLayout, icons } from '@facile/muse';
	import type { AuthUser } from '$lib/backend';
	import { useT, type Key } from '$lib/i18n';

	const t = useT();

	let { children } = $props();

	const auth = getContext<{ user: AuthUser | null }>('auth');

	const sections: { id: string; key: Key; icon: string; adminOnly?: boolean }[] = [
		{ id: 'profile', key: 'settings.profile', icon: icons.userCircle },
		{ id: 'preferences', key: 'settings.preferences', icon: icons.palette },
		{ id: 'api', key: 'settings.api', icon: icons.key, adminOnly: true },
		{ id: 'advanced', key: 'settings.advanced', icon: icons.settings }
	];

	const items = $derived(
		sections
			.filter((section) => !section.adminOnly || auth?.user?.is_admin)
			.map((section) => ({
				id: section.id,
				label: t(section.key),
				icon: section.icon,
				href: section.id === 'profile' ? '/settings' : `/settings/${section.id}`
			}))
	);
	const active = $derived(items.find((item) => item.href === (page.url.pathname.replace(/\/$/, '') || '/settings'))?.id);
</script>

<svelte:head><title>{t('settings.title')} — Journal</title></svelte:head>

<SettingsLayout sections={items} {active} backHref="/settings" navLabel={t('settings.sections')}>
	{@render children()}
</SettingsLayout>
