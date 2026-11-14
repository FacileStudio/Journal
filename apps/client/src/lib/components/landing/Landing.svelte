<script lang="ts">
	import {
		FeatureSplit,
		ForDevelopers,
		LANDING_HEADING,
		LandingFooter,
		LandingHeader,
		PricingPlans,
		SignInWithFacile,
		TextElevate,
		WorksWith,
		getI18n,
		icons,
		type DeveloperRow,
		type FeatureItem,
		type WorksWithApp
	} from '@facile/muse';
	import { ssoLoginUrl } from '$lib/backend';
	import { JOURNAL_ICON } from '$lib/icons';
	import { useT } from '$lib/i18n';
	import AlertTile from './AlertTile.svelte';
	import LandingActions, { OPEN_TO_NEW_ACCOUNTS, SUITE_PRICING } from './LandingActions.svelte';
	import LogDemo from './LogDemo.svelte';
	import SearchTile from './SearchTile.svelte';
	import StackTile from './StackTile.svelte';

	const t = useT();
	const i18n = getI18n();

	$effect(() => {
		document.querySelector('meta[name="description"]')?.setAttribute('content', t(OPEN_TO_NEW_ACCOUNTS ? 'landing.meta' : 'landing.metaSoon'));
	});

	const REPO = 'https://github.com/FacileStudio/Journal';
	const width = 'mx-auto w-full max-w-5xl px-4 sm:px-6';

	const features: FeatureItem[] = $derived([
		{ icon: icons.server, title: t('landing.features.collect.title'), text: t('landing.features.collect.body') },
		{ icon: icons.search, title: t('landing.features.search.title'), text: t('landing.features.search.body') },
		{ icon: icons.history, title: t('landing.features.tail.title'), text: t('landing.features.tail.body') },
		{ icon: icons.shield, title: t('landing.features.spaces.title'), text: t('landing.features.spaces.body') }
	]);

	const apps: WorksWithApp[] = $derived([{ name: 'Antenne', icon: icons.notification, line: t('landing.with.antenne') }]);

	const notes = $derived(
		OPEN_TO_NEW_ACCOUNTS
			? { solo: t('landing.pricing.free'), '*': t('landing.pricing.included') }
			: { solo: t('landing.pricing.freeSoon'), '*': t('landing.pricing.soon') }
	);

	const developers: DeveloperRow[] = $derived([
		{ kind: 'cli', text: t('landing.dev.cli'), href: '/downloads', command: 'brew install --cask FacileStudio/tap/journal' },
		{ kind: 'api', text: t('landing.dev.api'), href: `${REPO}/blob/main/docs/api.md` },
		{ kind: 'mcp', text: t('landing.dev.mcp'), href: `${REPO}/blob/main/docs/mcp.md` },
		{ kind: 'source', text: t('landing.dev.source'), href: REPO }
	]);

	const links = $derived([
		{ href: '#features', label: t('landing.nav.features') },
		{ href: '#pricing', label: t('landing.nav.pricing') },
		{ href: '#developers', label: t('landing.nav.developers') }
	]);

	const columns = $derived([
		{
			title: t('landing.footer.product'),
			links: [
				{ href: '#features', label: t('landing.nav.features') },
				{ href: '/downloads', label: t('landing.footer.downloads') },
				{ href: `${REPO}/tree/main/docs`, label: t('landing.footer.docs'), external: true }
			]
		}
	]);
</script>

<svelte:head>
	<title>{t('landing.title')}</title>
</svelte:head>

<div class="min-h-dvh overflow-x-clip bg-fc-page text-fc-fg">
	<LandingHeader appName="Journal" icon={JOURNAL_ICON} {links} skipHref="#main">
		{#snippet actions()}
			<SignInWithFacile href={ssoLoginUrl} size="sm" />
		{/snippet}
	</LandingHeader>

	<main id="main">
		<section class="mx-auto max-w-5xl px-4 pt-20 pb-12 text-center sm:px-6 sm:pt-24">
			<h1 class="font-fc-title text-fc-3xl font-bold tracking-tight sm:text-5xl">
				<TextElevate text={t('landing.hero.title')} delay={0.1} />
			</h1>
			<p class="mx-auto mt-6 max-w-xl text-fc-md text-fc-fg-muted">{t('landing.hero.body')}</p>
			<div class="mt-10">
				<LandingActions />
			</div>
			{#if !OPEN_TO_NEW_ACCOUNTS}
				<p class="mt-4 text-fc-sm text-fc-fg-muted">{t('landing.hero.soon')}</p>
			{/if}
		</section>

		<section class="mx-auto max-w-5xl px-4 pb-8 sm:px-6">
			<LogDemo />
		</section>

		<section class="relative mx-auto max-w-5xl px-4 py-16 sm:px-6 sm:py-20">
			<h2 class="mb-10 text-center {LANDING_HEADING}">{t('landing.cards.title')}</h2>
			<div class="grid gap-6 md:grid-cols-3">
				<SearchTile />
				<AlertTile />
				<StackTile />
			</div>
		</section>

		<FeatureSplit
			id="features"
			class="{width} scroll-mt-20 py-12"
			title={t('landing.features.heading')}
			description={t('landing.features.intro')}
			{features}
		/>

		<WorksWith class="{width} py-12" {apps} description={t('landing.with.line')} />

		<PricingPlans
			id="pricing"
			class="{width} scroll-mt-20 py-12"
			title={t('landing.pricing.title')}
			headingClass={LANDING_HEADING}
			highlightOnly
			{notes}
			recommended="pro"
			suiteHref={SUITE_PRICING}
			suiteNote={t('landing.pricing.suite')}
		/>

		<div id="developers" class="{width} scroll-mt-20 py-12">
			<ForDevelopers rows={developers} description={t('landing.dev.line')} selfHostHref={`https://suite.facile.studio/${i18n.locale === 'fr' ? '' : `${i18n.locale}/`}auto-hebergement`} />
		</div>

		<section class="mx-auto max-w-5xl px-4 py-20 text-center sm:px-6">
			<h2 class={LANDING_HEADING}>{t('landing.cta.title')}</h2>
			<p class="mt-4 text-fc-sm text-fc-fg-muted">{t(OPEN_TO_NEW_ACCOUNTS ? 'landing.cta.open' : 'landing.hero.soon')}</p>
			<div class="mt-8">
				<LandingActions />
			</div>
		</section>
	</main>

	<LandingFooter appName="Journal" icon={JOURNAL_ICON} tagline={t('app.tagline')} {columns} />
</div>
