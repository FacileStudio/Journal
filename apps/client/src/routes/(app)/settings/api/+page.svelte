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
	import { useT } from '$lib/i18n';

	const t = useT();

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
			error = err instanceof Error ? err.message : t('keys.loadSettingsFailed');
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
			toast.success(t('antenne.updated'));
		} catch (err) {
			toast.danger(err instanceof Error ? err.message : t('antenne.updateFailed'));
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
			createError = t('keys.checkFields');
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
			createError = err instanceof Error ? err.message : t('keys.createErrorTitle');
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
			toast.success(t('keys.revoked', { app: key.app }));
		} catch (err) {
			toast.danger(err instanceof Error ? err.message : t('keys.revokeFailed'));
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
	<SettingsSection title={t('keys.endpoints')} description={t('keys.endpointsHint')}>
		<SettingsRow
			label={t('keys.servers')}
			description={t('keys.serversHint')}
			stacked
		>
			<SecretField value={ingestUrl} sensitive={false} />
		</SettingsRow>
		<SettingsRow
			label={t('keys.browsers')}
			description={t('keys.browsersHint')}
			stacked
		>
			<SecretField value={browserUrl} sensitive={false} />
		</SettingsRow>
	</SettingsSection>

	<SettingsSection
		title={t('keys.title')}
		description={t('keys.titleHint')}
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
				{t('keys.new')}
			</Button>
		{/snippet}

		{#if error}
			<Alert tone="danger" title={t('keys.loadFailed')}>{error}</Alert>
		{:else if loading}
			<div class="flex justify-center py-16"><Spinner /></div>
		{:else if keys.length === 0}
			<EmptyState
				icon={icons.key}
				title={t('keys.empty')}
				description={t('keys.emptyHint')}
			>
				<Button icon={icons.plus} onclick={() => (open = true)}>{t('keys.new')}</Button>
			</EmptyState>
		{:else}
			<Table>
				<thead>
					<tr>
						<th scope="col">{t('common.app')}</th>
						<th scope="col">{t('keys.kind')}</th>
						<th scope="col" class="hidden sm:table-cell">{t('keys.prefix')}</th>
						<th scope="col">{t('keys.scope')}</th>
						<th scope="col" class="hidden sm:table-cell">{t('common.created')}</th>
						<th scope="col">{t('common.status')}</th>
						<th scope="col" aria-label={t('common.actions')}></th>
					</tr>
				</thead>
				<tbody>
					{#each keys as key (key.id)}
						<!-- Revoked keys stay listed: the audit trail has to keep naming them. -->
						<tr class={key.revoked_at ? 'opacity-55' : ''}>
							<td class="font-fc-mono text-fc-xs">{key.app}</td>
							<td>
								<Badge tone={key.kind === 'public' ? 'warning' : 'neutral'}>
									{key.kind === 'public' ? t('keys.public') : t('keys.secret')}
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
											{t('keys.usedToday', { used: key.used_today, quota: key.daily_quota })}
										</span>
									</div>
								{:else}
									<span class="text-fc-xs">{t('keys.serversNoQuota')}</span>
								{/if}
							</td>
							<td class="hidden whitespace-nowrap text-fc-fg-muted sm:table-cell">{formatDate(key.created_at)}</td>
							<td class="whitespace-nowrap text-fc-fg-muted">
								{key.revoked_at ? t('keys.revokedOn', { date: formatDate(key.revoked_at) }) : t('keys.active')}
							</td>
							<td class="text-right">
								{#if !key.revoked_at}
									<Button
										size="sm"
										variant="ghost-danger"
										icon={icons.revoke}
										onclick={() => (pending = key)}
									>
										{t('keys.revoke')}
									</Button>
								{/if}
							</td>
						</tr>
					{/each}
				</tbody>
			</Table>
		{/if}
	</SettingsSection>

	<SettingsSection title={t('antenne.title')} description={t('antenne.hint')}>
		{#if error}
			<Alert tone="danger" title={t('antenne.loadFailed')}>{error}</Alert>
		{:else if loading}
			<div class="flex justify-center py-16"><Spinner /></div>
		{:else}
			<SettingsRow
				label={t('antenne.enabled')}
				description={t('antenne.enabledHint')}
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
				label={t('antenne.url')}
				description={t('antenne.urlHint')}
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
				label={t('antenne.secret')}
				description={t('antenne.secretHint')}
				stacked
			>
				<SecretField
					bind:value={antenneSecret}
					label={t('antenne.secretLabel')}
					onchange={updateAntenneSettings}
					editable
					mask="full"
				/>
			</SettingsRow>
			<SettingsRow label={t('common.status')} stacked>
				{#if antenneConnected}
					<span class="flex items-center gap-2">
						<StatusDot tone="success" />
						<span class="font-fc-mono text-fc-xs">{t('antenne.connected')}</span>
					</span>
				{:else if antenneConnectError}
					<span class="flex items-center gap-2">
						<StatusDot tone="danger" />
						<span class="font-fc-mono text-fc-xs">{t('antenne.error', { message: antenneConnectError })}</span>
					</span>
				{:else}
					<span class="flex items-center gap-2">
						<StatusDot tone="warning" />
						<span class="font-fc-mono text-fc-xs">{t('antenne.disconnected')}</span>
					</span>
				{/if}
			</SettingsRow>
		{/if}
	</SettingsSection>
</div>

<Drawer
	bind:open
	title={issued ? t('keys.created') : t('keys.newTitle')}
	description={issued ? undefined : t('keys.newHint')}
	onClose={reset}
>
	{#if issued}
		<div class="flex flex-col gap-4">
			{#if issued.kind === 'public'}
				<Alert tone="info" title={t('keys.publicAlertTitle')}>
					{t('keys.publicAlertBody')}
				</Alert>
			{:else}
				<Alert tone="warning" title={t('keys.copyNowTitle')}>
					{t('keys.copyNowBody')}
				</Alert>
			{/if}
			<SecretField
				value={issued.token}
				label={t('keys.tokenFor', { app: issued.app })}
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
				<Alert tone="danger" title={t('keys.createErrorTitle')}>{createError}</Alert>
			{/if}

			<OptionCards
				label={t('keys.kind')}
				bind:value={kind}
				options={[
					{ value: 'secret', label: t('keys.optionSecret'), icon: icons.server },
					{ value: 'public', label: t('keys.optionPublic'), icon: icons.globe }
				]}
			/>

			<Field
				label={t('keys.appName')}
				helper={t('keys.appNameHint')}
				error={appName && !appNameValid ? t('keys.appNameError') : undefined}
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
					label={t('keys.origins')}
					helper={t('keys.originsHint')}
					error={originsText && !originsValid ? t('keys.originsError') : undefined}
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
					label={t('keys.quota')}
					helper={t('keys.quotaHint')}
					error={quota && !quotaValid ? t('keys.quotaError') : undefined}
				>
					<Input bind:value={quota} class="font-fc-mono" inputmode="numeric" autocomplete="off" />
				</Field>
			{/if}
		</div>
	{/if}

	{#snippet footer()}
		<div class="flex justify-end gap-2">
			{#if issued}
				<Button size="lg" onclick={() => (open = false)}>{t('common.done')}</Button>
			{:else}
				<Button size="lg" variant="ghost" onclick={() => (open = false)}>{t('common.cancel')}</Button>
				<Button size="lg" disabled={creating || !canCreate} onclick={create}>
					{creating ? t('common.creating') : t('keys.createKey')}
				</Button>
			{/if}
		</div>
	{/snippet}
</Drawer>

<ConfirmModal
	open={pending !== null}
	tone="danger"
	icon={icons.revoke}
	title={t('keys.revokeTitle')}
	description={t('keys.revokeBody')}
	confirmLabel={t('keys.revoke')}
	onConfirm={revoke}
	onCancel={() => (pending = null)}
/>
