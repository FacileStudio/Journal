<script lang="ts">
	import { getContext } from 'svelte';
	import { page } from '$app/state';
	import { Divider, PageTransition, Tabs, icons } from '@facile/muse';
	import type { AuthUser } from '$lib/backend';
	import PageHeader from '$lib/components/PageHeader.svelte';

	let { children } = $props();

	const auth = getContext<{ user: AuthUser | null }>('auth');

	const sections = [
		{ id: '/settings', label: 'Profile', icon: icons.userCircle },
		{ id: '/settings/appearance', label: 'Appearance', icon: icons.palette },
		{ id: '/settings/api', label: 'API', icon: icons.key, adminOnly: true },
		{ id: '/settings/advanced', label: 'Advanced', icon: icons.settings }
	];

	const items = $derived(
		sections
			.filter((section) => !section.adminOnly || auth?.user?.is_admin)
			.map((section) => ({
				id: section.id,
				label: section.label,
				icon: section.icon,
				href: section.id
			}))
	);
	const active = $derived(page.url.pathname.replace(/\/$/, '') || '/settings');
</script>

<svelte:head><title>Settings — Journal</title></svelte:head>

<PageHeader title="Settings" />

<!-- `gap-4`: pulled tighter the strip reads as an underline welded to the active pill. -->
<div class="flex flex-col gap-4">
	<Tabs {items} value={active} label="Settings sections" />
	<Divider />
</div>

<PageTransition key={active}>
	{@render children()}
</PageTransition>
