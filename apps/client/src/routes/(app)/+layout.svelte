<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { MobileNav, PageTransition, SideBar, Spinner, Topbar, icons } from '@facile/muse';
	import { backend, type AuthUser } from '$lib/backend';
	import { clearToken } from '$lib/auth';
	import { setContext } from 'svelte';

	let { children } = $props();

	let user = $state<AuthUser | null>(null);
	let ready = $state(false);
	let collapsed = $state(false);
	let scroller = $state<HTMLElement | null>(null);

	/*
	 * No Settings row: settings is reached from the user card at the bottom of the rail and from
	 * the avatar in MobileNav, and log out lives inside it. See muse CHARTE §14.
	 */
	const links = [
		{ label: 'Overview', href: '/', icon: icons.dashboard },
		{ label: 'Logs', href: '/logs', icon: icons.history },
		{ label: 'Apps', href: '/apps', icon: icons.server },
		{ label: 'Queries', href: '/queries', icon: icons.filter },
		{ label: 'Alerts', href: '/alerts', icon: icons.notification, adminOnly: true }
	];

	async function logout() {
		try {
			await backend.logout();
		} catch {
			/* token already gone or network hiccup — clear locally regardless */
		}
		clearToken();
		await goto('/login');
	}

	setContext('auth', {
		get user() {
			return user;
		},
		logout
	});

	/* A single sign-on session lives in an HttpOnly cookie, so there is no token here to
	   check and no way to read one. The API is the only thing that can answer whether this
	   browser is signed in, so ask it rather than guessing from localStorage. */
	$effect(() => {
		backend
			.me()
			.then((res) => {
				user = res.user;
				ready = true;
			})
			.catch(() => {
				clearToken();
				void goto('/login');
			});
	});

	/* <main> is the scroll container and sits outside PageTransition, so its scrollTop survives
	   a route change unless someone puts it back. */
	$effect(() => {
		if (page.url.pathname) scroller?.scrollTo({ top: 0 });
	});

	function isActive(href: string) {
		if (href === '/') return page.url.pathname === '/';
		return page.url.pathname === href || page.url.pathname.startsWith(`${href}/`);
	}

	const navPages = $derived(
		links
			.filter((link) => !link.adminOnly || user?.is_admin)
			.map((link) => ({ label: link.label, href: link.href, icon: link.icon, active: isActive(link.href) }))
	);
	const onSettings = $derived(page.url.pathname.startsWith('/settings'));

	/* Every settings section collapses to one key: the sections have their own PageTransition
	   inside the settings layout, and keying this one on the full path replays both. */
	const routeKey = $derived(onSettings ? '/settings' : page.url.pathname);
	const navUser = $derived.by(() => ({ name: user?.name || user?.email || 'Account' }));
</script>

{#if ready}
	<div class="flex h-dvh w-full overflow-hidden bg-fc-page">
		<div class="hidden h-full shrink-0 p-3 md:block">
			<SideBar
				icon="solar:notebook-bold-duotone"
				title="Journal"
				bind:collapsed
				pages={navPages}
				user={navUser}
				userHref="/settings"
				userActive={onSettings}
				class="h-full"
			/>
		</div>

		<!-- `min-w-0` so the log table scrolls itself instead of pushing the shell sideways, and
		     `overscroll-contain` because <main> is the only scroller. -->
		<main
			bind:this={scroller}
			class="min-w-0 flex-1 overflow-auto overscroll-contain pb-28 md:pb-0"
		>
			<Topbar class="md:hidden">
				<span class="text-fc-md font-semibold text-fc-fg">Journal</span>
			</Topbar>

			<div class="mx-auto flex max-w-fc-xl flex-col gap-10 px-4 py-8 sm:px-6 md:px-10 md:py-10">
				<PageTransition key={routeKey}>
					{@render children()}
				</PageTransition>
			</div>
		</main>

		<MobileNav items={navPages} user={navUser} profileHref="/settings" profileActive={onSettings} />
	</div>
{:else}
	<div class="flex h-dvh w-full items-center justify-center bg-fc-page">
		<Spinner />
	</div>
{/if}
