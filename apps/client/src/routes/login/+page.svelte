<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { Alert, AuthScreen, Icon } from '@facile/muse';
	import { JOURNAL_ICON } from '$lib/icons';
	import { backend, ssoLoginUrl } from '$lib/backend';
	import { rememberLocale } from '$lib/session.svelte';
	import { rememberReturn, safePath } from '$lib/return-path';
	import { useT } from '$lib/i18n';

	const t = useT();

	let refused = $state('');

	onMount(async () => {
		const query = new URLSearchParams(location.search);
		refused = query.get('error') ?? '';
		rememberReturn(query.get('next'));
		/* A single sign-on session is an HttpOnly cookie, so there is nothing here to
		   inspect: the API is the only thing that can say whether this browser is already
		   signed in. */
		try {
			await backend.me();
			void goto(safePath(new URLSearchParams(location.search).get('next')) ?? '/');
		} catch {
			return;
		}
	});
</script>

<svelte:head>
	<title>{t('login.title')} — {t('app.titleSuffix')}</title>
</svelte:head>

<AuthScreen appName="Journal" href={ssoLoginUrl} onLocaleChange={rememberLocale}>
	{#snippet logo()}
		<Icon icon={JOURNAL_ICON} size={32} />
	{/snippet}
	{#snippet tagline()}
		{t('app.tagline')}
	{/snippet}
	{#snippet notice()}
		{#if refused}
			<Alert tone="danger" title={refused} />
		{/if}
	{/snippet}
</AuthScreen>
