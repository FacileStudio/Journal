<script lang="ts">
	import { PreferencesSection, type Locale, type Theme } from '@facile/muse';
	import { rememberLocale } from '$lib/session.svelte';
	import { setTheme, theme } from '$lib/theme.svelte';
	import { useT } from '$lib/i18n';

	const t = useT();
</script>

<svelte:head><title>{t('settings.preferences')} — Journal</title></svelte:head>

<PreferencesSection
	theme={theme.mode}
	onThemeChange={(next: Theme) => setTheme(next)}
	onLocaleChange={(next: Locale) => rememberLocale(next)}
/>
