<script lang="ts">
	import { Button, DownloadSection, Icon, LandingFooter, LandingHeader, SignInWithFacile, TextElevate, icons } from '@facile/muse';
	import { ssoLoginUrl } from '$lib/backend';
	import { CLI_CHECKSUMS_URL, CLI_RELEASES_URL, cliCommands } from '$lib/cli';
	import { JOURNAL_ICON } from '$lib/icons';
	import { useT } from '$lib/i18n';
	import LogDemo from './LogDemo.svelte';

	const t = useT();

	const features = $derived([
		{ icon: icons.server, title: t('landing.features.collect.title'), body: t('landing.features.collect.body') },
		{ icon: icons.search, title: t('landing.features.search.title'), body: t('landing.features.search.body') },
		{ icon: icons.history, title: t('landing.features.tail.title'), body: t('landing.features.tail.body') },
		{ icon: icons.notification, title: t('landing.features.alerts.title'), body: t('landing.features.alerts.body') }
	]);

	const links = $derived([
		{ href: '#features', label: t('landing.nav.features') },
		{ href: '#download', label: t('landing.nav.download') }
	]);

	const columns = $derived([
		{
			title: t('landing.footer.product'),
			links: [
				{ href: '#features', label: t('landing.nav.features') },
				{ href: '/downloads', label: t('landing.footer.downloads') },
				{ href: 'https://github.com/FacileStudio/Journal/tree/main/docs', label: t('landing.footer.docs'), external: true }
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
			<div class="mt-10 flex flex-wrap items-center justify-center gap-3">
				<SignInWithFacile href={ssoLoginUrl} />
				<Button variant="secondary" href="#download" icon={icons.code}>{t('landing.hero.cli')}</Button>
			</div>
		</section>

		<section class="mx-auto max-w-5xl px-4 pb-8 sm:px-6">
			<LogDemo />
		</section>

		<section id="features" class="mx-auto max-w-5xl scroll-mt-20 px-4 py-12 sm:px-6">
			<ul class="grid gap-4 sm:grid-cols-2">
				{#each features as feature (feature.title)}
					<li class="rounded-fc-lg bg-fc-component p-6">
						<div class="mb-4 flex size-10 items-center justify-center rounded-fc-md bg-fc-surface">
							<Icon icon={feature.icon} size={20} class="block shrink-0" />
						</div>
						<h2 class="text-fc-lg font-bold">{feature.title}</h2>
						<p class="mt-1.5 text-fc-sm text-fc-fg-muted">{feature.body}</p>
					</li>
				{/each}
			</ul>
		</section>

		<section id="download" class="mx-auto flex max-w-5xl scroll-mt-20 flex-col gap-8 px-4 py-20 sm:px-6">
			<div class="text-center">
				<h2 class="font-fc-title text-fc-2xl font-bold tracking-tight">{t('landing.download.heading')}</h2>
				<p class="mx-auto mt-3 max-w-md text-fc-sm text-fc-fg-muted">{t('landing.download.intro')}</p>
			</div>
			<DownloadSection
				commands={cliCommands(t)}
				releasesHref={CLI_RELEASES_URL}
				checksumsHref={CLI_CHECKSUMS_URL}
				headingLevel={3}
			/>
		</section>

		<section class="mx-auto max-w-5xl px-4 py-20 text-center sm:px-6">
			<h2 class="font-fc-title text-fc-2xl font-bold tracking-tight">{t('landing.cta.title')}</h2>
			<p class="mt-4 text-fc-sm text-fc-fg-muted">{t('landing.cta.body')}</p>
			<div class="mt-8 flex justify-center">
				<SignInWithFacile href={ssoLoginUrl} />
			</div>
		</section>
	</main>

	<LandingFooter appName="Journal" icon={JOURNAL_ICON} tagline={t('app.tagline')} {columns} />
</div>
