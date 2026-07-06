<script lang="ts">
	import { getContext, onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { backend, type ApiKey, type AuthUser } from '$lib/backend';

	const auth = getContext<{ user: AuthUser | null; logout: () => void }>('auth');

	const APP_NAME_PATTERN = /^[a-z0-9][a-z0-9-]{0,63}$/;

	let keys = $state<ApiKey[]>([]);
	let loading = $state(true);
	let error = $state('');
	let appName = $state('');
	let creating = $state(false);
	let createError = $state('');
	let createdToken = $state<string | null>(null);
	let createdApp = $state('');
	let copied = $state(false);
	let revokingId = $state<number | null>(null);

	const appNameValid = $derived(APP_NAME_PATTERN.test(appName.trim()));

	async function loadKeys() {
		loading = true;
		error = '';
		try {
			const res = await backend.listApiKeys();
			keys = res.keys;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load API keys';
		} finally {
			loading = false;
		}
	}

	async function createKey(event: Event) {
		event.preventDefault();
		const app = appName.trim();
		if (creating) return;
		if (!APP_NAME_PATTERN.test(app)) {
			createError = 'App name must match ^[a-z0-9][a-z0-9-]{0,63}$ (lowercase letters, digits, hyphens).';
			return;
		}
		creating = true;
		createError = '';
		try {
			const res = await backend.createApiKey(app);
			createdToken = res.token;
			createdApp = res.key.app;
			copied = false;
			appName = '';
			await loadKeys();
		} catch (err) {
			createError = err instanceof Error ? err.message : 'Failed to create API key';
		} finally {
			creating = false;
		}
	}

	async function copyToken() {
		if (!createdToken) return;
		try {
			await navigator.clipboard.writeText(createdToken);
			copied = true;
		} catch {
			copied = false;
		}
	}

	async function revokeKey(key: ApiKey) {
		if (revokingId != null) return;
		if (!confirm(`Revoke the API key for "${key.app}"? Apps using it will stop being able to ingest logs.`)) return;
		revokingId = key.id;
		error = '';
		try {
			await backend.revokeApiKey(key.id);
			await loadKeys();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to revoke API key';
		} finally {
			revokingId = null;
		}
	}

	function formatDate(iso: string): string {
		const date = new Date(iso);
		if (Number.isNaN(date.getTime())) return iso;
		return date.toLocaleString(undefined, {
			year: 'numeric',
			month: 'short',
			day: '2-digit',
			hour: '2-digit',
			minute: '2-digit',
			hour12: false
		});
	}

	onMount(() => {
		if (!auth?.user?.is_admin) {
			goto('/');
			return;
		}
		loadKeys();
	});
</script>

<svelte:head>
	<title>Journal — API keys</title>
</svelte:head>

<div class="flex min-h-screen flex-col bg-background text-foreground">
	<header class="flex items-center gap-3 border-b border-border px-5 py-3">
		<a
			href="/"
			class="inline-flex h-9 items-center gap-2 rounded-md border border-border bg-background px-3 text-sm font-medium transition-colors hover:bg-accent"
		>
			<iconify-icon icon="solar:arrow-left-linear" width="16"></iconify-icon>
			Logs
		</a>
		<div class="flex items-center gap-2">
			<iconify-icon icon="solar:key-linear" width="18" class="text-foreground"></iconify-icon>
			<h1 class="text-lg font-bold font-heading tracking-tight">API keys</h1>
		</div>
		<div class="ml-auto flex items-center gap-2">
			{#if auth?.user}
				<span class="hidden max-w-[12rem] truncate text-sm text-muted-foreground sm:inline" title={auth.user.email}>
					{auth.user.name || auth.user.email}
				</span>
			{/if}
			<button
				class="inline-flex h-9 w-9 items-center justify-center rounded-md border border-border bg-background transition-colors hover:bg-accent"
				title="Sign out"
				aria-label="Sign out"
				onclick={() => auth?.logout()}
			>
				<iconify-icon icon="solar:logout-2-linear" width="16"></iconify-icon>
			</button>
		</div>
	</header>

	{#if error}
		<div class="border-b border-border bg-destructive/10 px-5 py-2 text-sm text-destructive">{error}</div>
	{/if}

	<main class="mx-auto w-full max-w-3xl flex-1 px-5 py-6">
		<section class="rounded-xl border border-border bg-card p-5 shadow-sm">
			<h2 class="text-sm font-semibold">Create a key</h2>
			<p class="mt-1 text-sm text-muted-foreground">
				Per-app ingest keys. Each key is scoped to one app name and can be revoked independently.
			</p>

			<form class="mt-4 flex flex-col gap-3 sm:flex-row sm:items-start" onsubmit={createKey}>
				<div class="flex flex-1 flex-col gap-1.5">
					<input
						type="text"
						bind:value={appName}
						placeholder="app name, e.g. nuage"
						autocomplete="off"
						spellcheck="false"
						class="h-11 rounded-md border border-input bg-background px-3 font-mono text-sm placeholder:font-sans placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
					/>
					{#if appName.trim() && !appNameValid}
						<span class="text-xs text-destructive">Lowercase letters, digits, and hyphens only; must start with a letter or digit; max 64 characters.</span>
					{/if}
				</div>
				<button
					type="submit"
					disabled={creating || !appNameValid}
					class="inline-flex h-11 items-center justify-center rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground transition-opacity hover:opacity-90 disabled:opacity-50"
				>
					{creating ? 'Creating…' : 'Create key'}
				</button>
			</form>

			{#if createError}
				<p class="mt-3 rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">{createError}</p>
			{/if}

			{#if createdToken}
				<div class="mt-4 rounded-md border border-border bg-muted/40 p-4">
					<div class="flex items-center justify-between gap-2">
						<p class="text-sm font-medium">Token for <span class="font-mono">{createdApp}</span></p>
						<button
							class="inline-flex h-9 w-9 items-center justify-center rounded-md border border-border bg-background transition-colors hover:bg-accent"
							title="Dismiss"
							aria-label="Dismiss token"
							onclick={() => (createdToken = null)}
						>
							<iconify-icon icon="solar:close-circle-linear" width="16"></iconify-icon>
						</button>
					</div>
					<div class="mt-2 flex items-center gap-2">
						<code class="min-w-0 flex-1 overflow-x-auto whitespace-nowrap rounded-md border border-border bg-background px-3 py-2.5 font-mono text-xs">{createdToken}</code>
						<button
							class="inline-flex h-11 shrink-0 items-center gap-1.5 rounded-md border border-border bg-background px-3 text-sm font-medium transition-colors hover:bg-accent"
							onclick={copyToken}
						>
							<iconify-icon icon={copied ? 'solar:check-circle-bold' : 'solar:copy-linear'} width="16"></iconify-icon>
							{copied ? 'Copied' : 'Copy'}
						</button>
					</div>
					<p class="mt-2 text-xs text-destructive">You won't see this token again — copy it now.</p>
				</div>
			{/if}
		</section>

		<section class="mt-6 rounded-xl border border-border bg-card shadow-sm">
			<div class="border-b border-border px-5 py-3">
				<h2 class="text-sm font-semibold">Keys</h2>
			</div>
			{#if loading}
				<p class="px-5 py-8 text-center text-sm text-muted-foreground">Loading…</p>
			{:else if keys.length === 0}
				<p class="px-5 py-8 text-center text-sm text-muted-foreground">No API keys yet.</p>
			{:else}
				<ul class="divide-y divide-border">
					{#each keys as key (key.id)}
						<li class="flex flex-wrap items-center gap-3 px-5 py-3">
							<span class="rounded-md bg-secondary px-1.5 py-0.5 text-xs font-medium text-secondary-foreground">{key.app}</span>
							<code class="font-mono text-xs text-muted-foreground">{key.prefix}…</code>
							<span class="text-xs text-muted-foreground">created {formatDate(key.created_at)}</span>
							<span class="ml-auto flex items-center gap-2">
								{#if key.revoked_at}
									<span class="rounded-md bg-destructive/10 px-1.5 py-0.5 text-xs font-medium uppercase text-destructive">revoked</span>
								{:else}
									<button
										class="inline-flex h-11 items-center rounded-md border border-border bg-background px-3 text-xs font-medium text-destructive transition-colors hover:bg-destructive/10 disabled:opacity-50"
										onclick={() => revokeKey(key)}
										disabled={revokingId != null}
									>
										{revokingId === key.id ? 'Revoking…' : 'Revoke'}
									</button>
								{/if}
							</span>
						</li>
					{/each}
				</ul>
			{/if}
		</section>
	</main>
</div>
