<script lang="ts">
	import { getContext, onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { backend, type AlertRule, type AuthUser, type SavedQuery } from '$lib/backend';

	const auth = getContext<{ user: AuthUser | null; logout: () => void }>('auth');

	let alerts = $state<AlertRule[]>([]);
	let savedQueries = $state<SavedQuery[]>([]);
	let loading = $state(true);
	let error = $state('');

	let name = $state('');
	let savedQueryId = $state('');
	let threshold = $state<number | null>(1);
	let windowMinutes = $state<number | null>(5);
	let webhookUrl = $state('');
	let webhookHeader = $state('');
	let webhookSecret = $state('');
	let creating = $state(false);
	let createError = $state('');
	let togglingId = $state<number | null>(null);
	let deletingId = $state<number | null>(null);

	const thresholdValid = $derived(threshold != null && Number.isInteger(threshold) && threshold >= 1);
	const windowValid = $derived(
		windowMinutes != null && Number.isInteger(windowMinutes) && windowMinutes >= 1 && windowMinutes <= 1440
	);
	const webhookValid = $derived(isHttpUrl(webhookUrl));
	const headerPairValid = $derived((webhookHeader.trim() === '') === (webhookSecret === ''));
	const formValid = $derived(
		name.trim().length > 0 && savedQueryId !== '' && thresholdValid && windowValid && webhookValid && headerPairValid
	);

	function isHttpUrl(value: string): boolean {
		try {
			const url = new URL(value.trim());
			return url.protocol === 'http:' || url.protocol === 'https:';
		} catch {
			return false;
		}
	}

	function webhookHost(value: string): string {
		try {
			return new URL(value).host;
		} catch {
			return value;
		}
	}

	async function loadAll() {
		loading = true;
		error = '';
		try {
			const [alertsRes, queriesRes] = await Promise.all([backend.listAlerts(), backend.listQueries()]);
			alerts = alertsRes.alerts;
			savedQueries = queriesRes.queries;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load alert rules';
		} finally {
			loading = false;
		}
	}

	async function createRule(event: Event) {
		event.preventDefault();
		if (creating || !formValid) return;
		creating = true;
		createError = '';
		try {
			await backend.createAlert({
				name: name.trim(),
				saved_query_id: Number(savedQueryId),
				threshold: threshold as number,
				window_minutes: windowMinutes as number,
				webhook_url: webhookUrl.trim(),
				...(webhookHeader.trim()
					? { webhook_header: webhookHeader.trim(), webhook_secret: webhookSecret }
					: {})
			});
			name = '';
			savedQueryId = '';
			threshold = 1;
			windowMinutes = 5;
			webhookUrl = '';
			webhookHeader = '';
			webhookSecret = '';
			await loadAll();
		} catch (err) {
			createError = err instanceof Error ? err.message : 'Failed to create alert rule';
		} finally {
			creating = false;
		}
	}

	async function toggleEnabled(rule: AlertRule) {
		if (togglingId != null) return;
		togglingId = rule.id;
		error = '';
		try {
			await backend.updateAlert(rule.id, { enabled: !rule.enabled });
			await loadAll();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to update alert rule';
		} finally {
			togglingId = null;
		}
	}

	async function deleteRule(rule: AlertRule) {
		if (deletingId != null) return;
		if (!confirm(`Delete alert rule "${rule.name}"? It will stop firing to its webhook.`)) return;
		deletingId = rule.id;
		error = '';
		try {
			await backend.deleteAlert(rule.id);
			await loadAll();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to delete alert rule';
		} finally {
			deletingId = null;
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
		loadAll();
	});
</script>

<svelte:head>
	<title>Journal — Alerts</title>
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
			<iconify-icon icon="solar:bell-linear" width="18" class="text-foreground"></iconify-icon>
			<h1 class="text-lg font-bold font-heading tracking-tight">Alerts</h1>
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
			<h2 class="text-sm font-semibold">Create a rule</h2>
			<p class="mt-1 text-sm text-muted-foreground">
				A rule counts matches of a saved query over a rolling window and POSTs to a webhook when the count reaches the threshold.
			</p>

			{#if !loading && savedQueries.length === 0}
				<div class="mt-4 rounded-md border border-border bg-muted/40 px-4 py-3 text-sm text-muted-foreground">
					No saved queries yet. Alert rules are built on saved queries —
					<a href="/" class="font-medium text-foreground underline underline-offset-2 hover:opacity-80">save one on the dashboard</a>
					first.
				</div>
			{:else}
				<form class="mt-4 grid gap-3 sm:grid-cols-2" onsubmit={createRule}>
					<label class="flex flex-col gap-1.5 text-xs font-medium text-muted-foreground">
						Name
						<input
							type="text"
							bind:value={name}
							placeholder="e.g. nuage errors spike"
							autocomplete="off"
							class="h-11 rounded-md border border-input bg-background px-3 text-sm text-foreground placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
						/>
					</label>
					<label class="flex flex-col gap-1.5 text-xs font-medium text-muted-foreground">
						Saved query
						<select
							bind:value={savedQueryId}
							class="h-11 rounded-md border border-input bg-background px-3 text-sm text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
						>
							<option value="" disabled>Select a saved query…</option>
							{#each savedQueries as saved (saved.id)}
								<option value={String(saved.id)}>{saved.name}</option>
							{/each}
						</select>
					</label>
					<label class="flex flex-col gap-1.5 text-xs font-medium text-muted-foreground">
						Threshold (count ≥)
						<input
							type="number"
							min="1"
							step="1"
							bind:value={threshold}
							class="h-11 rounded-md border border-input bg-background px-3 text-sm text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
						/>
						{#if threshold != null && !thresholdValid}
							<span class="text-xs font-normal text-destructive">Must be a whole number ≥ 1.</span>
						{/if}
					</label>
					<label class="flex flex-col gap-1.5 text-xs font-medium text-muted-foreground">
						Window (minutes)
						<input
							type="number"
							min="1"
							max="1440"
							step="1"
							bind:value={windowMinutes}
							class="h-11 rounded-md border border-input bg-background px-3 text-sm text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
						/>
						{#if windowMinutes != null && !windowValid}
							<span class="text-xs font-normal text-destructive">Must be between 1 and 1440 minutes.</span>
						{/if}
					</label>
					<label class="flex flex-col gap-1.5 text-xs font-medium text-muted-foreground sm:col-span-2">
						Webhook URL
						<input
							type="url"
							bind:value={webhookUrl}
							placeholder="https://nook.facile.studio/hooks/…"
							autocomplete="off"
							spellcheck="false"
							class="h-11 rounded-md border border-input bg-background px-3 font-mono text-sm text-foreground placeholder:font-sans placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
						/>
						{#if webhookUrl.trim() && !webhookValid}
							<span class="text-xs font-normal text-destructive">Must be a valid http(s) URL.</span>
						{/if}
					</label>
					<label class="flex flex-col gap-1.5 text-xs font-medium text-muted-foreground">
						Auth header name (optional)
						<input
							type="text"
							bind:value={webhookHeader}
							placeholder="e.g. X-Webhook-Token"
							autocomplete="off"
							spellcheck="false"
							class="h-11 rounded-md border border-input bg-background px-3 font-mono text-sm text-foreground placeholder:font-sans placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
						/>
					</label>
					<label class="flex flex-col gap-1.5 text-xs font-medium text-muted-foreground">
						Auth header secret
						<input
							type="password"
							bind:value={webhookSecret}
							placeholder="secret value"
							autocomplete="off"
							class="h-11 rounded-md border border-input bg-background px-3 font-mono text-sm text-foreground placeholder:font-sans placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
						/>
						{#if !headerPairValid}
							<span class="text-xs font-normal text-destructive">Header name and secret go together — fill both or neither.</span>
						{/if}
					</label>
					<div class="sm:col-span-2">
						<button
							type="submit"
							disabled={creating || !formValid}
							class="inline-flex h-11 items-center justify-center rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground transition-opacity hover:opacity-90 disabled:opacity-50"
						>
							{creating ? 'Creating…' : 'Create rule'}
						</button>
					</div>
				</form>
			{/if}

			{#if createError}
				<p class="mt-3 rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">{createError}</p>
			{/if}
		</section>

		<section class="mt-6 rounded-xl border border-border bg-card shadow-sm">
			<div class="border-b border-border px-5 py-3">
				<h2 class="text-sm font-semibold">Rules</h2>
			</div>
			{#if loading}
				<p class="px-5 py-8 text-center text-sm text-muted-foreground">Loading…</p>
			{:else if alerts.length === 0}
				<p class="px-5 py-8 text-center text-sm text-muted-foreground">No alert rules yet.</p>
			{:else}
				<ul class="divide-y divide-border">
					{#each alerts as rule (rule.id)}
						<li class="flex flex-wrap items-center gap-3 px-5 py-3">
							<div class="min-w-0 flex-1">
								<div class="flex flex-wrap items-center gap-2">
									<span class="text-sm font-medium">{rule.name}</span>
									<span class="rounded-md bg-secondary px-1.5 py-0.5 text-xs font-medium text-secondary-foreground">{rule.query_name}</span>
									<span class="text-xs text-muted-foreground">count ≥ {rule.threshold} in {rule.window_minutes}m</span>
								</div>
								<div class="mt-1 flex flex-wrap items-center gap-3 text-xs text-muted-foreground">
									<span class="inline-flex items-center gap-1" title={rule.webhook_url}>
										<iconify-icon icon="solar:link-linear" width="12"></iconify-icon>
										{webhookHost(rule.webhook_url)}
									</span>
									<span>last fired {rule.last_fired_at ? formatDate(rule.last_fired_at) : 'never'}</span>
								</div>
							</div>
							<span class="ml-auto flex items-center gap-2">
								<button
									class="inline-flex h-11 items-center gap-1.5 rounded-md border border-border px-3 text-xs font-medium transition-colors disabled:opacity-50 {rule.enabled ? 'bg-primary text-primary-foreground hover:opacity-90' : 'bg-background text-muted-foreground hover:bg-accent'}"
									aria-pressed={rule.enabled}
									onclick={() => toggleEnabled(rule)}
									disabled={togglingId != null}
								>
									<span class="inline-block h-2 w-2 rounded-full {rule.enabled ? 'bg-green-500' : 'bg-muted-foreground'}"></span>
									{togglingId === rule.id ? 'Saving…' : rule.enabled ? 'Enabled' : 'Disabled'}
								</button>
								<button
									class="inline-flex h-11 items-center rounded-md border border-border bg-background px-3 text-xs font-medium text-destructive transition-colors hover:bg-destructive/10 disabled:opacity-50"
									onclick={() => deleteRule(rule)}
									disabled={deletingId != null}
								>
									{deletingId === rule.id ? 'Deleting…' : 'Delete'}
								</button>
							</span>
						</li>
					{/each}
				</ul>
			{/if}
		</section>
	</main>
</div>
