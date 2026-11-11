<script lang="ts">
	import '../app.css';
	import { browser } from '$app/environment';
	import { Toaster, provideLocale } from '@facile/muse';
	import { registerAppIcons } from '$lib/icons';
	import { theme } from '$lib/theme.svelte';
	import { session } from '$lib/session.svelte';

	let { children } = $props();

	provideLocale(() => session.locale);

	registerAppIcons();

	if (browser) {
		theme.restore();
		forgetLegacyToken();
	}

	/* The dashboard once kept a bearer session token in localStorage. The session is the HttpOnly
	   cookie now and nothing reads the key, so a stored token is only a credential left lying in
	   script-readable storage. */
	function forgetLegacyToken() {
		try {
			localStorage.removeItem('journal.token');
		} catch {
			return;
		}
	}
</script>

{@render children()}

<!-- One Toaster for the whole app, outside the router, so a navigation cannot unmount a
     toast mid-flight. The extra bottom padding clears MobileNav on a phone. -->
<Toaster class="pb-28 md:pb-6" />
