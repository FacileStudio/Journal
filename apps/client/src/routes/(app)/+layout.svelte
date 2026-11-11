<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { MobileNav, PageTransition, SideBar, Spinner, Topbar, icons, resolveLocale, type Locale, type Theme } from '@facile/muse';
	import { backend, type AuthUser } from '$lib/backend';
	import { JOURNAL_ICON } from '$lib/icons';
	import { adoptLocale, rememberLocale, session } from '$lib/session.svelte';
	import { theme } from '$lib/theme.svelte';
	import { useT, type Key } from '$lib/i18n';
	import { takeReturn } from '$lib/return-path';
	import { setContext } from 'svelte';

	let { children } = $props();

	const t = useT();

	let user = $state<AuthUser | null>(null);
	let invitations = $state<{ count: number; href: string } | null>(null);
	let ready = $state(false);
	let collapsed = $state(false);
	let scroller = $state<HTMLElement | null>(null);

	const links: { key: Key; href: string; icon: string; adminOnly?: boolean }[] = [
		{ key: 'nav.overview', href: '/', icon: icons.dashboard },
		{ key: 'nav.logs', href: '/logs', icon: icons.history },
		{ key: 'nav.apps', href: '/apps', icon: icons.server },
		{ key: 'nav.queries', href: '/queries', icon: icons.filter },
		{ key: 'nav.alerts', href: '/alerts', icon: icons.notification, adminOnly: true }
	];

	async function logout() {
		try {
			await backend.logout();
		} catch {
			/* token already gone or network hiccup — clear locally regardless */
		}
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
				adoptLocale(resolveLocale(res.user.locale));
				ready = true;
				loadInvitations();
				const back = takeReturn();
				if (back) void goto(back, { replaceState: true });
			})
			.catch(() => {
				const next = page.url.pathname === '/' ? '' : `?next=${encodeURIComponent(page.url.pathname + page.url.search)}`;
				void goto(`/login${next}`);
			});
	});

	/* Registre's pending space invitations for the user menu. An answer without href is no row,
	   and a failure is no row either: the menu never waits on Registre. */
	function loadInvitations() {
		backend
			.invitations()
			.then((answer) => (invitations = answer.href ? { count: answer.count, href: answer.href } : null))
			.catch(() => (invitations = null));
	}

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
			.map((link) => ({ label: t(link.key), href: link.href, icon: link.icon, active: isActive(link.href) }))
	);
	const onSettings = $derived(page.url.pathname.startsWith('/settings'));

	/* Every settings section collapses to one key: the sections have their own PageTransition
	   inside the settings layout, and keying this one on the full path replays both. */
	const routeKey = $derived(onSettings ? '/settings' : page.url.pathname);
	const navUser = $derived.by(() => ({
		name: user?.name || user?.email || t('nav.account'),
		email: user?.email,
		avatar: user?.avatar_url || undefined
	}));
	const userMenu = $derived({
		subscription: { plan: 'Gratuit', href: 'https://sso.facile.studio/spaces' },
		helpHref: 'https://github.com/FacileStudio/Journal/tree/main/docs',
		onSignOut: logout,
		locale: session.locale,
		onLocaleChange: (next: Locale) => rememberLocale(next),
		theme: theme.mode,
		onThemeChange: (next: Theme) => theme.set(next),
		invitations
	});
</script>

{#if ready}
	<div class="flex h-dvh w-full overflow-hidden bg-fc-page">
		<div class="hidden h-full shrink-0 p-3 md:block">
			<SideBar
				icon={JOURNAL_ICON}
				title="Journal"
				bind:collapsed
				pages={navPages}
				user={navUser}
				userHref="/settings"
				userActive={onSettings}
				{userMenu}
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

			<!-- The gap belongs on PageTransition, not on the wrapper: PageTransition
			     is the wrapper's only child, so a gap there separates nothing and every
			     page rendered its heading flush against its own content. -->
			<div class="mx-auto max-w-fc-xl px-4 py-8 sm:px-6 md:px-10 md:py-10">
				<PageTransition key={routeKey} class="flex flex-col gap-10">
					{@render children()}
				</PageTransition>
			</div>
		</main>

		<MobileNav items={navPages} user={navUser} profileHref="/settings" profileActive={onSettings} {userMenu} />
	</div>
{:else}
	<div class="flex h-dvh w-full items-center justify-center bg-fc-page">
		<Spinner />
	</div>
{/if}
