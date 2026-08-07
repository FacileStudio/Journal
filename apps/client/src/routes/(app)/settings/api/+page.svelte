<script lang="ts">
	import { getContext, onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import {
		Alert,
		Button,
		ConfirmModal,
		Drawer,
		EmptyState,
		Field,
		Input,
		SecretField,
		SettingsRow,
		SettingsSection,
		Spinner,
		Table,
		icons,
		toast
	} from '@facile/muse';
	import { backend, type ApiKey, type AuthUser } from '$lib/backend';
	import { formatDate } from '$lib/format';

	const APP_NAME = /^[a-z0-9][a-z0-9-]{0,63}$/;

	const auth = getContext<{ user: AuthUser | null }>('auth');

	let keys = $state<ApiKey[]>([]);
	let loading = $state(true);
	let error = $state('');

	let open = $state(false);
	let appName = $state('');
	let creating = $state(false);
	let createError = $state('');
	let issued = $state<{ app: string; token: string } | null>(null);

	let pending = $state<ApiKey | null>(null);

	const appNameValid = $derived(APP_NAME.test(appName.trim()));
	const ingestUrl = $derived(
		typeof window === 'undefined' ? '/api/ingest' : `${window.location.origin}/api/ingest`
	);

	async function load() {
		try {
			keys = (await backend.listApiKeys()).keys;
			error = '';
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load API keys';
		} finally {
			loading = false;
		}
	}

	/* Reopening must never resurrect the previous token — it is shown exactly once. */
	function reset() {
		appName = '';
		createError = '';
		issued = null;
	}

	async function create() {
		if (!appNameValid) {
			createError = 'App name must match ^[a-z0-9][a-z0-9-]{0,63}$.';
			return;
		}
		creating = true;
		createError = '';
		try {
			const res = await backend.createApiKey(appName.trim());
			issued = { app: res.key.app, token: res.token };
			await load();
		} catch (err) {
			createError = err instanceof Error ? err.message : 'Could not create the key';
		} finally {
			creating = false;
		}
	}

	async function revoke() {
		const key = pending;
		if (!key) return;
		try {
			await backend.revokeApiKey(key.id);
			await load();
			toast.success(`Revoked the key for ${key.app}.`);
		} catch (err) {
			toast.danger(err instanceof Error ? err.message : 'Could not revoke the key');
		} finally {
			pending = null;
		}
	}

	onMount(() => {
		if (!auth?.user?.is_admin) {
			void goto('/settings');
			return;
		}
		void load();
	});
</script>

<div class="flex flex-col gap-10">
	<SettingsSection title="Ingest endpoint" description="Where shippers POST their entries.">
		<SettingsRow
			label="URL"
			description="The trailing /api is load-bearing — without it the dashboard's catch-all answers 200 and every line is discarded."
			stacked
		>
			<SecretField value={ingestUrl} sensitive={false} />
		</SettingsRow>
	</SettingsSection>

	<SettingsSection
		title="Ingest keys"
		description="One key is scoped to one app name. Multiple active keys per app are allowed, so rotation is add-new → redeploy → revoke-old with no downtime."
		bare
	>
		{#snippet actions()}
			<Button
				size="sm"
				icon={icons.plus}
				onclick={() => {
					reset();
					open = true;
				}}
			>
				New key
			</Button>
		{/snippet}

		{#if error}
			<Alert tone="danger" title="Could not load API keys">{error}</Alert>
		{:else if loading}
			<div class="flex justify-center py-16"><Spinner /></div>
		{:else if keys.length === 0}
			<EmptyState
				icon={icons.key}
				title="No ingest keys yet"
				description="Without a key and without the legacy INGEST_TOKEN, every POST to /api/ingest is rejected."
			>
				<Button icon={icons.plus} onclick={() => (open = true)}>New key</Button>
			</EmptyState>
		{:else}
			<Table>
				<thead>
					<tr>
						<th scope="col">App</th>
						<th scope="col">Prefix</th>
						<th scope="col">Created</th>
						<th scope="col">Status</th>
						<th scope="col" aria-label="Actions"></th>
					</tr>
				</thead>
				<tbody>
					{#each keys as key (key.id)}
						<!-- Revoked keys stay listed: the audit trail has to keep naming them. -->
						<tr class={key.revoked_at ? 'opacity-55' : ''}>
							<td class="font-fc-mono text-fc-xs">{key.app}</td>
							<td class="font-fc-mono text-fc-xs text-fc-fg-muted">{key.prefix}…</td>
							<td class="whitespace-nowrap text-fc-fg-muted">{formatDate(key.created_at)}</td>
							<td class="whitespace-nowrap text-fc-fg-muted">
								{key.revoked_at ? `revoked ${formatDate(key.revoked_at)}` : 'active'}
							</td>
							<td class="text-right">
								{#if !key.revoked_at}
									<Button
										size="sm"
										variant="ghost-danger"
										icon={icons.revoke}
										onclick={() => (pending = key)}
									>
										Revoke
									</Button>
								{/if}
							</td>
						</tr>
					{/each}
				</tbody>
			</Table>
		{/if}
	</SettingsSection>
</div>

<Drawer
	bind:open
	title={issued ? 'Key created' : 'New ingest key'}
	description={issued
		? undefined
		: 'The app name scopes the key: entries it ships must carry this name or none at all.'}
	onClose={reset}
>
	{#if issued}
		<div class="flex flex-col gap-4">
			<Alert tone="warning" title="Copy it now">
				This is the only time the token is shown. Journal stores nothing but its SHA-256.
			</Alert>
			<SecretField value={issued.token} label="Token for {issued.app}" autoHideMs={0} />
		</div>
	{:else}
		<div class="flex flex-col gap-4">
			{#if createError}
				<Alert tone="danger" title="Could not create the key">{createError}</Alert>
			{/if}
			<Field
				label="App name"
				helper="Lowercase letters, digits and hyphens."
				error={appName && !appNameValid ? 'Must match ^[a-z0-9][a-z0-9-]{0,63}$.' : undefined}
			>
				<Input
					bind:value={appName}
					class="font-fc-mono"
					placeholder="nuage"
					autocomplete="off"
					spellcheck={false}
				/>
			</Field>
		</div>
	{/if}

	{#snippet footer()}
		<div class="flex justify-end gap-2">
			{#if issued}
				<Button size="lg" onclick={() => (open = false)}>Done</Button>
			{:else}
				<Button size="lg" variant="ghost" onclick={() => (open = false)}>Cancel</Button>
				<Button size="lg" disabled={creating || !appNameValid} onclick={create}>
					{creating ? 'Creating…' : 'Create key'}
				</Button>
			{/if}
		</div>
	{/snippet}
</Drawer>

<ConfirmModal
	open={pending !== null}
	tone="danger"
	icon={icons.revoke}
	title="Revoke this ingest key?"
	description="Any pipeline still using it starts failing immediately, and a revoked key cannot be un-revoked — issue a new one and redeploy instead."
	confirmLabel="Revoke"
	onConfirm={revoke}
	onCancel={() => (pending = null)}
/>
