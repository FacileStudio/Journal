<script lang="ts">
	import { onMount, setContext } from 'svelte';
	import { goto } from '$app/navigation';
	import { backend, type AuthUser } from '$lib/backend';
	import { clearToken, getToken } from '$lib/auth';

	let { children } = $props();

	let user = $state<AuthUser | null>(null);
	let ready = $state(false);

	async function logout() {
		try {
			await backend.logout();
		} catch {
			/* token already gone or network hiccup — clear locally regardless */
		}
		clearToken();
		goto('/login');
	}

	setContext('auth', {
		get user() {
			return user;
		},
		logout
	});

	onMount(async () => {
		if (!getToken()) {
			goto('/login');
			return;
		}
		try {
			const res = await backend.me();
			user = res.user;
			ready = true;
		} catch {
			clearToken();
			goto('/login');
		}
	});
</script>

{#if ready}
	{@render children()}
{:else}
	<div class="flex h-screen items-center justify-center bg-background text-sm text-muted-foreground">
		Loading…
	</div>
{/if}
