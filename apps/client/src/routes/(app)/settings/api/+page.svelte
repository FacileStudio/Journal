<script lang="ts">
	import { getContext, onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import {
		Alert,
		Badge,
		Button,
		ConfirmModal,
		Drawer,
		EmptyState,
		Field,
		Input,
		OptionCards,
		SecretField,
		SettingsRow,
		SettingsSection,
		Spinner,
		StatusDot,
		Switch,
		Table,
		Textarea,
		icons,
		toast
	} from '@facile/muse';
	import { backend, type ApiKey, type ApiKeyKind, type AuthUser } from '$lib/backend';
	import { formatDate } from '$lib/format';

	const APP_NAME = /^[a-z0-9][a-z0-9-]{0,63}$/;
	const ORIGIN = /^https?:\/\/[^/?#\s]+$/;
	const DEFAULT_QUOTA = 10000;

	const auth = getContext<{ user: AuthUser | null }>('auth');

	let keys = $state<ApiKey[]>([]);
	let loading = $state(true);
	let error = $state('');
	let antenneLoading = $state(false);
	let antenneUrl = $state('');
	let antenneSecret = $state('');
	let antenneEnabled = $state(false);
	let antenneConnected = $state(false);
	let antenneConnectError = $state('');

	let open = $state(false);
	let appName = $state('');
	let kind = $state<ApiKeyKind>('secret');
	let originsText = $state('');
	let quota = $state(String(DEFAULT_QUOTA));
	let creating = $state(false);
	let createError = $state('');
	let issued = $state<{ app: string; kind: ApiKeyKind; token: string } | null>(null);

	let pending = $state<ApiKey | null>(null);

	const appNameValid = $derived(APP_NAME.test(appName.trim()));
	const origins = $derived(
		originsText
			.split(/[\n,]/)
			.map((line) => line.trim())
			.filter(Boolean)
	);
	const originsValid = $derived(
		kind === 'secret' || (origins.length > 0 && origins.length <= 8 && origins.every((o) => ORIGIN.test(o)))
	);
	const quotaValid = $derived(kind === 'secret' || (Number(quota) >= 1 && Number(quota) <= 10_000_000));
	const canCreate = $derived(appNameValid && originsValid && quotaValid);

	const baseUrl = $derived(typeof window === 'undefined' ? '' : window.location.origin);
	const ingestUrl = $derived(`${baseUrl}/api/ingest`);
	const browserUrl = $derived(`${baseUrl}/api/ingest/browser`);

	/*
	 * The snippet is the whole point of showing the token: a public key is
	 * useless until it is wired into hooks.client.ts, and pasting it wrong is
	 * how the reports end up in nobody's dashboard.
	 */
	/* Every origin is https and the scheme eats a third of the column on a phone,
	   so the host carries the meaning and the full value stays in the title. */
	function hostOf(origin: string): string {
		return origin.replace(/^https?:\/\//, '');
	}

	const snippet = $derived(
		issued
			? `import { createJournal } from '@facile/journal';
import { handleErrorWith } from '@facile/journal/sveltekit';

const journal = createJournal({
	url: '${baseUrl}/api',
	key: '${issued.token}'
});

journal.install();
export const handleError = handleErrorWith(journal);`
			: ''
	);

	async function load() {
		try {
			const [keysRes, antenneRes] = await Promise.all([
				backend.listApiKeys(),
				backend.getAntenneSettings()
			]);
			keys = keysRes.keys;
			const { settings } = antenneRes;
			antenneUrl = settings.url;
			antenneSecret = settings.secret;
			antenneEnabled = settings.enabled;
			antenneConnected = settings.connected;
			antenneConnectError = settings.connect_error ?? '';
			error = '';
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load settings';
		} finally {
			loading = false;
			antenneLoading = false;
		}
	}

	async function updateAntenneSettings() {
		antenneLoading = true;
		try {
			const { settings } = await backend.updateAntenneSettings({
				url: antenneUrl,
				secret: antenneSecret,
				enabled: antenneEnabled
			});
			antenneUrl = settings.url;
			antenneSecret = settings.secret;
			antenneEnabled = settings.enabled;
			antenneConnected = settings.connected;
			antenneConnectError = settings.connect_error ?? '';
			toast.success('Antenne settings updated.');
		} catch (err) {
			toast.danger(err instanceof Error ? err.message : 'Failed to update Antenne settings');
		} finally {
			antenneLoading = false;
		}
	}

	/* Reopening must never resurrect the previous token — it is shown exactly once. */
	function reset() {
		appName = '';
		kind = 'secret';
		originsText = '';
		quota = String(DEFAULT_QUOTA);
		createError = '';
		issued = null;
	}

	async function create() {
		if (!canCreate) {
			createError = 'Check the app name, the origins and the quota.';
			return;
		}
		creating = true;
		createError = '';
		try {
			const res = await backend.createApiKey({
				app: appName.trim(),
				kind,
				allowed_origins: kind === 'public' ? origins : undefined,
				daily_quota: kind === 'public' ? Number(quota) : undefined
			});
			issued = { app: res.key.app, kind: res.key.kind, token: res.token };
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

<div class="flex min-w-0 flex-col gap-10">
	<SettingsSection title="Ingest endpoints" description="Where shippers and pages send their entries.">
		<SettingsRow
			label="Servers"
			description="The trailing /api is load-bearing — without it the dashboard's catch-all answers 200 and every line is discarded."
			stacked
		>
			<SecretField value={ingestUrl} sensitive={false} />
		</SettingsRow>
		<SettingsRow
			label="Browsers"
			description="Authenticated by a public key, restricted to that key's origins, and capped by its daily quota. Used by the @facile/journal SDK."
			stacked
		>
			<SecretField value={browserUrl} sensitive={false} />
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
						<th scope="col">Kind</th>
						<th scope="col" class="hidden sm:table-cell">Prefix</th>
						<th scope="col">Scope</th>
						<th scope="col" class="hidden sm:table-cell">Created</th>
						<th scope="col">Status</th>
						<th scope="col" aria-label="Actions"></th>
					</tr>
				</thead>
				<tbody>
					{#each keys as key (key.id)}
						<!-- Revoked keys stay listed: the audit trail has to keep naming them. -->
						<tr class={key.revoked_at ? 'opacity-55' : ''}>
							<td class="font-fc-mono text-fc-xs">{key.app}</td>
							<td>
								<Badge tone={key.kind === 'public' ? 'warning' : 'neutral'}>
									{key.kind === 'public' ? 'public' : 'secret'}
								</Badge>
							</td>
							<td class="hidden font-fc-mono text-fc-xs text-fc-fg-muted sm:table-cell">{key.prefix}…</td>
							<td class="max-w-[11rem] text-fc-fg-muted sm:max-w-none">
								{#if key.kind === 'public'}
									<div class="flex flex-col gap-1">
										{#each key.allowed_origins as origin (origin)}
											<span class="truncate font-fc-mono text-fc-xs" title={origin}>{hostOf(origin)}</span>
										{/each}
										<span class="text-fc-xs">
											{key.used_today.toLocaleString()} / {key.daily_quota.toLocaleString()} today
										</span>
									</div>
								{:else}
									<span class="text-fc-xs">servers, no quota</span>
								{/if}
							</td>
							<td class="hidden whitespace-nowrap text-fc-fg-muted sm:table-cell">{formatDate(key.created_at)}</td>
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

	<SettingsSection
		title="Antenne"
		description="Configure the global Antenne instance for alert delivery."
	>
		{#if error}
			<Alert tone="danger" title="Could not load Antenne settings">{error}</Alert>
		{:else if loading}
			<div class="flex justify-center py-16"><Spinner /></div>
		{:else}
			<SettingsRow
				label="Enabled"
				description="When enabled, alerts will be sent to the configured Antenne instance."
				stacked
			>
				<Switch
					checked={antenneEnabled}
					onchange={() => {
						antenneEnabled = !antenneEnabled;
						void updateAntenneSettings();
					}}
				/>
			</SettingsRow>
			<SettingsRow
				label="Instance URL"
				description="The URL of your Antenne instance (e.g., https://antenne.facile.studio)"
				stacked
			>
				<Input
					bind:value={antenneUrl}
					class="font-fc-mono"
					placeholder="https://antenne.facile.studio"
					onchange={updateAntenneSettings}
				/>
			</SettingsRow>
			<SettingsRow
				label="Secret"
				description="The secret key for authenticating with Antenne"
				stacked
			>
				<SecretField
					bind:value={antenneSecret}
					label="Antenne Secret"
					onchange={updateAntenneSettings}
					editable
					mask="full"
				/>
			</SettingsRow>
			<SettingsRow
				label="Status"
				stacked
			>
				{#if antenneConnected}
					<span class="flex items-center gap-2">
						<StatusDot tone="success" />
						<span class="font-fc-mono text-fc-xs">Connected</span>
					</span>
				{:else if antenneConnectError}
					<span class="flex items-center gap-2">
						<StatusDot tone="danger" />
						<span class="font-fc-mono text-fc-xs">Error: {antenneConnectError}</span>
					</span>
				{:else}
					<span class="flex items-center gap-2">
						<StatusDot tone="warning" />
						<span class="font-fc-mono text-fc-xs">Disconnected</span>
					</span>
				{/if}
			</SettingsRow>
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
			{#if issued.kind === 'public'}
				<Alert tone="info" title="This one belongs in your bundle">
					A public key is meant to be shipped to browsers. What protects Journal is the origin
					allowlist, the rate limit and the daily quota — not the secrecy of this string.
				</Alert>
			{:else}
				<Alert tone="warning" title="Copy it now">
					This is the only time the token is shown. Journal stores nothing but its SHA-256.
				</Alert>
			{/if}
			<SecretField
				value={issued.token}
				label="Token for {issued.app}"
				sensitive={issued.kind === 'secret'}
				autoHideMs={0}
			/>
			{#if issued.kind === 'public'}
				<Field label="src/hooks.client.ts" helper="bun add github:FacileStudio/Journal#ts">
					<Textarea value={snippet} rows={11} readonly class="font-fc-mono text-fc-xs" />
				</Field>
			{/if}
		</div>
	{:else}
		<div class="flex flex-col gap-4">
			{#if createError}
				<Alert tone="danger" title="Could not create the key">{createError}</Alert>
			{/if}

			<OptionCards
				label="Kind"
				bind:value={kind}
				options={[
					{ value: 'secret', label: 'Secret — servers', icon: icons.server },
					{ value: 'public', label: 'Public — browsers', icon: icons.globe }
				]}
			/>

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

			{#if kind === 'public'}
				<Field
					label="Allowed origins"
					helper="One per line, scheme://host[:port], no path and no wildcard. Up to 8. Anything else is refused."
					error={originsText && !originsValid
						? 'Each line must look like https://shop.example or http://localhost:5173, 8 at most.'
						: undefined}
				>
					<Textarea
						bind:value={originsText}
						rows={3}
						class="font-fc-mono text-fc-xs"
						placeholder={'https://shop.example\nhttp://localhost:5173'}
						spellcheck={false}
					/>
				</Field>

				<Field
					label="Daily quota"
					helper="Entries accepted per UTC day. The bound that holds if this key is abused — set it to a few times what a normal day looks like."
					error={quota && !quotaValid ? 'Between 1 and 10000000.' : undefined}
				>
					<Input bind:value={quota} class="font-fc-mono" inputmode="numeric" autocomplete="off" />
				</Field>
			{/if}
		</div>
	{/if}

	{#snippet footer()}
		<div class="flex justify-end gap-2">
			{#if issued}
				<Button size="lg" onclick={() => (open = false)}>Done</Button>
			{:else}
				<Button size="lg" variant="ghost" onclick={() => (open = false)}>Cancel</Button>
				<Button size="lg" disabled={creating || !canCreate} onclick={create}>
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
