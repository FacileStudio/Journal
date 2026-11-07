<script lang="ts">
	import { getContext } from 'svelte';
	import { page } from '$app/state';
	import { Divider, PageTransition, Tabs, icons } from '@facile/muse';
	import type { AuthUser } from '$lib/backend';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import { useT, type Key } from '$lib/i18n';

	const t = useT();

	let { children } = $props();

	const auth = getContext<{ user: AuthUser | null }>('auth');

	const sections: { id: string; key: Key; icon: string; adminOnly?: boolean }[] = [
		{ id: '/settings', key: 'settings.profile', icon: icons.userCircle },
		{ id: '/settings/appearance', key: 'settings.appearance', icon: icons.palette },
		{ id: '/settings/api', key: 'settings.api', icon: icons.key, adminOnly: true },
		{ id: '/settings/advanced', key: 'settings.advanced', icon: icons.settings }
	];

	const items = $derived(
		sections
			.filter((section) => !section.adminOnly || auth?.user?.is_admin)
			.map((section) => ({
				id: section.id,
				label: t(section.key),
				icon: section.icon,
				href: section.id
			}))
	);
	const active = $derived(page.url.pathname.replace(/\/$/, '') || '/settings');
</script>

<svelte:head><title>{t('settings.title')} — Journal</title></svelte:head>

<PageHeader title={t('settings.title')} />

<!-- `gap-4`: pulled tighter the strip reads as an underline welded to the active pill. -->
<div class="flex flex-col gap-4">
	<Tabs {items} value={active} label={t('settings.sections')} />
	<Divider />
</div>

<PageTransition key={active}>
	{@render children()}
</PageTransition>
