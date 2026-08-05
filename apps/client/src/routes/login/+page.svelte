<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { backend } from '$lib/backend';
	import { clearToken, getToken, setToken } from '$lib/auth';

	const inputClass =
		'flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50';
	const labelClass = 'text-sm font-medium leading-none';
	const primaryButtonClass =
		'inline-flex h-10 w-full items-center justify-center rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:pointer-events-none disabled:opacity-50';

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
	<title>{mode === 'register' ? 'Create account' : 'Log in'} — Journal</title>
</svelte:head>

<div class="flex min-h-screen">
	<div class="hidden lg:flex lg:w-1/2 flex-col bg-black px-12 py-10">
		<a href="/" class="flex items-center gap-3 mb-auto">
			<iconify-icon icon="solar:notebook-bold-duotone" width="28" class="text-white"></iconify-icon>
			<span class="text-xl font-bold font-heading tracking-tight text-white">Journal</span>
		</a>

		<div class="mb-auto">
			<h2 class="text-4xl font-bold font-heading text-white leading-tight tracking-tight">
				Your logs.<br />Your server.
			</h2>
			<p class="mt-4 text-sm text-white/50 max-w-xs leading-relaxed">
				Centralized logging for every app in the Facile Suite.
			</p>
		</div>

		<p class="text-xs text-white/30">
			© {new Date().getFullYear()} Journal by Facile.
		</p>
	</div>

	<div class="flex w-full lg:w-1/2 flex-col items-center justify-center px-8 py-12 bg-background">
		<div class="w-full max-w-sm">
			<div class="mb-8">
				<h1 class="text-2xl font-bold font-heading tracking-tight text-foreground">
					{mode === 'register' ? 'Create account' : 'Welcome back'}
				</h1>
				<p class="mt-1.5 text-sm text-muted-foreground">
					{mode === 'register'
						? 'Set up the first account to access the logs.'
						: 'Log in to your Journal account.'}
				</p>
			</div>

			{#if allowRegistration}
				<div class="mb-6 flex rounded-lg border border-border bg-muted p-1 gap-1" role="tablist">
					<button
						type="button"
						role="tab"
						aria-selected={mode === 'login'}
						class="flex-1 rounded-md py-1.5 text-sm font-medium transition-colors {mode === 'login'
							? 'bg-background text-foreground shadow-sm'
							: 'text-muted-foreground hover:text-foreground'}"
						onclick={() => switchMode('login')}
					>Log in</button>
					<button
						type="button"
						role="tab"
						aria-selected={mode === 'register'}
						class="flex-1 rounded-md py-1.5 text-sm font-medium transition-colors {mode === 'register'
							? 'bg-background text-foreground shadow-sm'
							: 'text-muted-foreground hover:text-foreground'}"
						onclick={() => switchMode('register')}
					>Register</button>
				</div>
			{/if}

			<form class="space-y-4" onsubmit={submit}>
				{#if mode === 'register'}
					<div class="space-y-1.5">
						<label for="name" class={labelClass}>Name</label>
						<input
							id="name"
							type="text"
							bind:value={name}
							placeholder="Ada Lovelace"
							autocomplete="name"
							disabled={busy}
							class={inputClass}
						/>
					</div>
				{/if}

				<div class="space-y-1.5">
					<label for="email" class={labelClass}>Email</label>
					<input
						id="email"
						type="email"
						bind:value={email}
						placeholder="you@example.com"
						autocomplete="email"
						required
						disabled={busy}
						class={inputClass}
					/>
				</div>

				<div class="space-y-1.5">
					<label for="password" class={labelClass}>Password</label>
					<input
						id="password"
						type="password"
						bind:value={password}
						placeholder="••••••••"
						autocomplete={mode === 'register' ? 'new-password' : 'current-password'}
						minlength={mode === 'register' ? 12 : undefined}
						required
						disabled={busy}
						class={inputClass}
					/>
					{#if mode === 'register'}
						<p class="text-xs text-muted-foreground">At least 12 characters.</p>
					{/if}
				</div>

				{#if error}
					<p class="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">
						{error}
					</p>
				{/if}

				<button type="submit" disabled={busy} class={primaryButtonClass}>
					{busy
						? mode === 'register' ? 'Creating account…' : 'Logging in…'
						: mode === 'register' ? 'Create account' : 'Log in'}
				</button>
			</form>
		</div>
	</div>
</div>
