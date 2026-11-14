<script lang="ts">
	import { DownloadSection, LandingFooter, LandingHeader, SignInWithFacile } from '@facile/muse';
	import { ssoLoginUrl } from '$lib/backend';
	import { CLI_CHECKSUMS_URL, CLI_RELEASES_URL, cliCommands } from '$lib/cli';
	import { JOURNAL_ICON } from '$lib/icons';
	import { useT } from '$lib/i18n';

	const t = useT();
</script>

<svelte:head>
	<title>{t('downloads.title')} — {t('app.titleSuffix')}</title>
</svelte:head>

<div class="flex min-h-dvh flex-col overflow-x-clip bg-fc-page text-fc-fg">
	<LandingHeader appName="Journal" icon={JOURNAL_ICON} skipHref="#main">
		{#snippet actions()}
			<SignInWithFacile href={ssoLoginUrl} size="sm" />
		{/snippet}
	</LandingHeader>

	<main id="main" class="mx-auto flex w-full max-w-5xl flex-1 flex-col gap-8 px-4 py-16 sm:px-6">
		<div class="text-center">
			<h1 class="font-fc-title text-fc-3xl font-bold tracking-tight">{t('downloads.title')}</h1>
			<p class="mx-auto mt-3 max-w-md text-fc-sm text-fc-fg-muted">{t('downloads.intro')}</p>
		</div>
		<DownloadSection commands={cliCommands(t)} releasesHref={CLI_RELEASES_URL} checksumsHref={CLI_CHECKSUMS_URL} />
	</main>

	<LandingFooter appName="Journal" icon={JOURNAL_ICON} tagline={t('app.tagline')} />
</div>
