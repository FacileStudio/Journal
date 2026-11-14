<script module lang="ts">
	/** Flip once Registre lets every account into Journal (access_policy no longer `assigned`). */
	export const OPEN_TO_NEW_ACCOUNTS = false;

	export const SUITE_PRICING = 'https://suite.facile.studio/#tarifs';
</script>

<script lang="ts">
	import { Button, LandingCta, SignInWithFacile } from '@facile/muse';
	import { ssoLoginUrl } from '$lib/backend';
	import { useT } from '$lib/i18n';

	interface Props {
		align?: 'start' | 'center';
	}

	let { align = 'center' }: Props = $props();

	const t = useT();
</script>

{#if OPEN_TO_NEW_ACCOUNTS}
	<LandingCta signInHref={ssoLoginUrl} {align} />
{:else}
	<div class={['flex flex-col gap-3 sm:flex-row sm:flex-wrap sm:items-center', align === 'center' && 'sm:justify-center']}>
		<Button href={SUITE_PRICING} size="lg" class="h-12 max-sm:w-full">{t('landing.hero.offers')}</Button>
		<SignInWithFacile href={ssoLoginUrl} size="lg" variant="secondary" class="max-sm:w-full" />
	</div>
{/if}
