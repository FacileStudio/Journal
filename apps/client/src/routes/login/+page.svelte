<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { backend } from '$lib/backend';
	import { clearToken, getToken, setToken } from '$lib/auth';

	let mode = $state<'login' | 'register'>('login');
	let email = $state('');
	let password = $state('');
	let name = $state('');
	let allowRegistration = $state(false);
	let busy = $state(false);
	let error = $state('');

	onMount(async () => {
		if (getToken()) {
			try {
				await backend.me();
				goto('/');
				return;
			} catch {
				clearToken();
			}
		}
		try {
			const cfg = await backend.authConfig();
			allowRegistration = cfg.allow_registration;
		} catch {
			allowRegistration = false;
		}
	});

	function switchMode(next: 'login' | 'register') {
		mode = next;
		error = '';
	}

	async function submit(event: Event) {
		event.preventDefault();
		if (busy) return;
		busy = true;
		error = '';
		try {
			const res =
				mode === 'register'
					? await backend.register(email, password, name)
					: await backend.login(email, password);
			setToken(res.token);
			goto('/');
		} catch (err) {
			error = err instanceof Error ? err.message : 'Something went wrong';
		} finally {
			busy = false;
		}
	}
</script>

<svelte:head>
	<title>Journal — Sign in</title>
</svelte:head>

<div class="flex min-h-screen items-center justify-center bg-background px-4 text-foreground">
	<div class="w-full max-w-sm">
		<div class="mb-8 flex items-center justify-center gap-2">
			<iconify-icon icon="solar:notebook-bold-duotone" width="26" class="text-foreground"></iconify-icon>
			<span class="text-2xl font-bold font-heading tracking-tight">Journal</span>
		</div>

		<div class="rounded-xl border border-border bg-card p-6 shadow-sm">
			<h1 class="text-lg font-semibold">
				{mode === 'register' ? 'Create your account' : 'Welcome back'}
			</h1>
			<p class="mt-1 text-sm text-muted-foreground">
				{mode === 'register'
					? 'Set up the first account to access the logs.'
					: 'Sign in to view the Suite logs.'}
			</p>

			<form class="mt-6 flex flex-col gap-4" onsubmit={submit}>
				{#if mode === 'register'}
					<label class="flex flex-col gap-1.5 text-sm">
						<span class="font-medium">Name</span>
						<input
							type="text"
							bind:value={name}
							autocomplete="name"
							placeholder="Ada Lovelace"
							class="h-10 rounded-md border border-input bg-background px-3 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
						/>
					</label>
				{/if}

				<label class="flex flex-col gap-1.5 text-sm">
					<span class="font-medium">Email</span>
					<input
						type="email"
						bind:value={email}
						required
						autocomplete="email"
						placeholder="you@facile.studio"
						class="h-10 rounded-md border border-input bg-background px-3 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
					/>
				</label>

				<label class="flex flex-col gap-1.5 text-sm">
					<span class="font-medium">Password</span>
					<input
						type="password"
						bind:value={password}
						required
						minlength={mode === 'register' ? 12 : undefined}
						autocomplete={mode === 'register' ? 'new-password' : 'current-password'}
						placeholder="••••••••••••"
						class="h-10 rounded-md border border-input bg-background px-3 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
					/>
					{#if mode === 'register'}
						<span class="text-xs text-muted-foreground">At least 12 characters.</span>
					{/if}
				</label>

				{#if error}
					<p class="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">{error}</p>
				{/if}

				<button
					type="submit"
					disabled={busy}
					class="inline-flex h-10 items-center justify-center rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground transition-opacity hover:opacity-90 disabled:opacity-50"
				>
					{busy ? 'Please wait…' : mode === 'register' ? 'Create account' : 'Sign in'}
				</button>
			</form>

			{#if allowRegistration}
				<p class="mt-4 text-center text-sm text-muted-foreground">
					{#if mode === 'login'}
						No account yet?
						<button class="font-medium text-foreground hover:underline" onclick={() => switchMode('register')}>
							Create one
						</button>
					{:else}
						Already have an account?
						<button class="font-medium text-foreground hover:underline" onclick={() => switchMode('login')}>
							Sign in
						</button>
					{/if}
				</p>
			{/if}
		</div>
	</div>
</div>
