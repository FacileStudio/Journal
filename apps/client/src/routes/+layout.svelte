<script lang="ts">
	import '../app.css';
	import { browser } from '$app/environment';
	import { Toaster, provideLocale } from '@facile/muse';
	import { applyStoredTheme } from '$lib/theme.svelte';
	import { session } from '$lib/session.svelte';

	let { children } = $props();

	provideLocale(() => session.locale);

	if (browser) {
		applyStoredTheme();
	}
</script>

{@render children()}

<!-- One Toaster for the whole app, outside the router, so a navigation cannot unmount a
     toast mid-flight. The extra bottom padding clears MobileNav on a phone. -->
<Toaster class="pb-28 md:pb-6" />
