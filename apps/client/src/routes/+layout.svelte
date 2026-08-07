<script lang="ts">
	import '../app.css';
	import { browser } from '$app/environment';
	import { Toaster } from '@facile/muse';
	import { applyStoredTheme } from '$lib/theme.svelte';

	let { children } = $props();

	/*
	 * muse renders <iconify-icon> custom elements and they stay inert until the element is
	 * registered. Importing the package here rather than pulling a CDN script keeps the
	 * dashboard self-hosted and satisfies the `script-src: self` CSP in svelte.config.js.
	 */
	if (browser) {
		void import('iconify-icon');
		applyStoredTheme();
	}
</script>

{@render children()}

<!-- One Toaster for the whole app, outside the router, so a navigation cannot unmount a
     toast mid-flight. The extra bottom padding clears MobileNav on a phone. -->
<Toaster class="pb-28 md:pb-6" />
