<script lang="ts">
	import { OptionCards, SettingsRow, SettingsSection, icons } from '@facile/muse';
	import { setTheme, theme, type ThemeMode } from '$lib/theme.svelte';
	import { useT } from '$lib/i18n';

	const t = useT();

	const modes = $derived([
		{ value: 'system', label: t('theme.system'), icon: icons.monitor },
		{ value: 'light', label: t('theme.light'), icon: icons.sun },
		{ value: 'dark', label: t('theme.dark'), icon: icons.moon }
	]);

	let mode = $state<string>(theme.mode);

	$effect(() => {
		setTheme(mode as ThemeMode);
	});
</script>

<div class="flex flex-col gap-10">
	<SettingsSection title={t('theme.title')} description={t('theme.hint')}>
		<SettingsRow
			label={t('theme.scheme')}
			description={t('theme.schemeHint')}
			stacked
		>
			<OptionCards options={modes} bind:value={mode} name="theme-mode" label={t('theme.scheme')} />
		</SettingsRow>
	</SettingsSection>
</div>
