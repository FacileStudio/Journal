<script lang="ts">
	import { getContext, onMount, untrack } from 'svelte';
	import { goto } from '$app/navigation';
	import {
		Alert,
		Button,
		ConfirmModal,
		Drawer,
		EmptyState,
		Field,
		Input,
		Select,
		SecretField,
		Spinner,
		Switch,
		Table,
		icons,
		toast
	} from '@facile/muse';
	import { backend, type AlertRule, type AuthUser, type SavedQuery, type AntenneSettings } from '$lib/backend';
	import { formatDate } from '$lib/format';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import SpaceFilter from '$lib/components/SpaceFilter.svelte';
	import SpaceTag from '$lib/components/SpaceTag.svelte';
	import { spaces } from '$lib/workspaces.svelte';
	import { useT } from '$lib/i18n';

	const t = useT();

	const auth = getContext<{ user: AuthUser | null }>('auth');

	let alerts = $state<AlertRule[]>([]);
	let queries = $state<SavedQuery[]>([]);
	let loading = $state(true);
	let error = $state('');
	let pending = $state<AlertRule | null>(null);
	let togglingId = $state<number | null>(null);

	let open = $state(false);
	let creating = $state(false);
	let createError = $state('');
	let name = $state('');
	let savedQueryId = $state('');
	let threshold = $state('1');
	let windowMinutes = $state('5');
	let webhookUrl = $state('');
	let webhookHeader = $state('');
	let webhookSecret = $state('');
	let provider = $state<'webhook' | 'antenne'>('webhook');
	const watchable = $derived(queries.filter((saved) => spaces.owned.some((w) => w.id === saved.workspace_id)));
	const antenneAllowed = $derived(watchable.find((saved) => String(saved.id) === savedQueryId)?.workspace_id === 1);

	function isHttpUrl(value: string): boolean {
		try {
			const url = new URL(value.trim());
			return url.protocol === 'http:' || url.protocol === 'https:';
		} catch {
			return false;
		}
	}

	function host(value: string): string {
		try {
			return new URL(value).host;
		} catch {
			return value;
		}
	}

	const thresholdNum = $derived(Number(threshold));
	const windowNum = $derived(Number(windowMinutes));
	const thresholdValid = $derived(Number.isInteger(thresholdNum) && thresholdNum >= 1);
	const windowValid = $derived(Number.isInteger(windowNum) && windowNum >= 1 && windowNum <= 1440);
	const webhookValid = $derived(isHttpUrl(webhookUrl));
	/* A header name without a secret sends an empty credential, and a secret without a name has
	   nowhere to go — the pair is all-or-nothing. */
	const headerPairValid = $derived((webhookHeader.trim() === '') === (webhookSecret === ''));
	const formValid = $derived(
		name.trim().length > 0 &&
			savedQueryId !== '' &&
			thresholdValid &&
			windowValid &&
			(provider === 'antenne' || (webhookValid && headerPairValid))
	);

	async function load() {
		try {
			const [alertsRes, queriesRes] = await Promise.all([
				backend.listAlerts(),
				backend.listQueries()
			]);
			alerts = alertsRes.alerts;
			queries = queriesRes.queries;
			error = '';
		} catch (err) {
			error = err instanceof Error ? err.message : t('alerts.loadFailed');
		} finally {
			loading = false;
		}
	}

	function reset() {
		name = '';
		savedQueryId = '';
		threshold = '1';
		windowMinutes = '5';
		webhookUrl = '';
		webhookHeader = '';
		webhookSecret = '';
		provider = 'webhook';
		createError = '';
	}

	async function create() {
		if (!formValid) return;
		creating = true;
		createError = '';
		try {
			await backend.createAlert({
				name: name.trim(),
				saved_query_id: Number(savedQueryId),
				provider: provider,
				threshold: thresholdNum,
				window_minutes: windowNum,
				webhook_url: webhookUrl.trim(),
				...(provider === 'webhook' && webhookHeader.trim()
					? { webhook_header: webhookHeader.trim(), webhook_secret: webhookSecret }
					: {})
			});
			await load();
			open = false;
			reset();
			toast.success(t('alerts.created'));
		} catch (err) {
			createError = err instanceof Error ? err.message : t('alerts.createFailed');
		} finally {
			creating = false;
		}
	}

	async function toggle(rule: AlertRule) {
		togglingId = rule.id;
		try {
			await backend.updateAlert(rule.id, { enabled: !rule.enabled });
			await load();
		} catch (err) {
			toast.danger(err instanceof Error ? err.message : t('alerts.updateFailed'));
		} finally {
			togglingId = null;
		}
	}

	async function remove() {
		const rule = pending;
		if (!rule) return;
		try {
			await backend.deleteAlert(rule.id);
			await load();
			toast.success(t('alerts.deleted', { name: rule.name }));
		} catch (err) {
			toast.danger(err instanceof Error ? err.message : t('alerts.deleteFailed'));
		} finally {
			pending = null;
		}
	}

	onMount(() => {
		if (spaces.owned.length === 0) {
			void goto('/');
			return;
		}
		void load();
	});

	$effect(() => {
		if (spaces.version) untrack(() => void load());
	});
</script>

<svelte:head><title>{t('alerts.title')} — Journal</title></svelte:head>

<PageHeader
	title={t('alerts.title')}
	description={t('alerts.description')}
>
	{#snippet actions()}
		<SpaceFilter />
		<Button
			size="sm"
			icon={icons.plus}
			disabled={watchable.length === 0}
			onclick={() => {
				reset();
				open = true;
			}}
		>
			{t('alerts.newRule')}
		</Button>
	{/snippet}
</PageHeader>

{#if error}
	<Alert tone="danger" title={t('alerts.loadFailed')}>{error}</Alert>
{:else if loading}
	<div class="flex justify-center py-16"><Spinner /></div>
{:else if watchable.length === 0}
	<EmptyState
		icon={icons.filter}
		title={t('alerts.saveQueryFirst')}
		description={t('alerts.saveQueryHint')}
	>
		<Button href="/logs" icon={icons.history}>{t('queries.openExplorer')}</Button>
	</EmptyState>
{:else if alerts.length === 0}
	<EmptyState
		icon={icons.notification}
		title={t('alerts.empty')}
		description={t('alerts.emptyHint')}
	>
		<Button icon={icons.plus} onclick={() => (open = true)}>{t('alerts.newRule')}</Button>
	</EmptyState>
{:else}
	<Table>
		<thead>
			<tr>
				<th scope="col">{t('alerts.rule')}</th>
				<th scope="col">{t('alerts.condition')}</th>
				<th scope="col">{t('alerts.target')}</th>
				<th scope="col">{t('alerts.lastFired')}</th>
				<th scope="col">{t('alerts.enabled')}</th>
				<th scope="col" aria-label={t('common.actions')}></th>
			</tr>
		</thead>
		<tbody>
			{#each alerts as rule (rule.id)}
				<tr>
					<td class="font-medium"><span class="inline-flex items-center gap-2">{rule.name}<SpaceTag id={rule.workspace_id} /></span></td>
					<td class="text-fc-fg-muted">
						<span class="font-fc-mono text-fc-xs">{rule.query_name}</span>
						· {t('alerts.conditionValue', { threshold: rule.threshold, minutes: rule.window_minutes })}
					</td>
					<td class="font-fc-mono text-fc-xs text-fc-fg-muted">
						{#if rule.provider === 'antenne'}
							{t('alerts.antenneGlobal')}
						{:else}
							{host(rule.webhook_url)}
						{/if}
					</td>
					<td class="whitespace-nowrap text-fc-fg-muted">
						{rule.last_fired_at ? formatDate(rule.last_fired_at) : t('common.never')}
					</td>
					<td>
						<Switch
							checked={rule.enabled}
							disabled={togglingId === rule.id}
							aria-label={t('alerts.enable', { name: rule.name })}
							onchange={() => toggle(rule)}
						/>
					</td>
					<td class="text-right">
						<Button
							size="sm"
							variant="ghost-danger"
							icon={icons.remove}
							onclick={() => (pending = rule)}
						>
							{t('common.delete')}
						</Button>
					</td>
				</tr>
			{/each}
		</tbody>
	</Table>
{/if}

<Drawer
	bind:open
	title={t('alerts.drawerTitle')}
	description={t('alerts.drawerHint')}
	onClose={reset}
>
	<div class="flex flex-col gap-4">
		{#if createError}
			<Alert tone="danger" title={t('alerts.createErrorTitle')}>{createError}</Alert>
		{/if}

		<Field label={t('common.name')}>
			<Input bind:value={name} placeholder={t('alerts.namePlaceholder')} />
		</Field>

		<Field label={t('alerts.savedQuery')} helper={t('alerts.savedQueryHint')}>
			<Select bind:value={savedQueryId}>
				<option value="" disabled>{t('alerts.pickQuery')}</option>
				{#each watchable as saved (saved.id)}
					<option value={String(saved.id)}>{spaces.multi ? `${saved.name} · ${spaces.name(saved.workspace_id)}` : saved.name}</option>
				{/each}
			</Select>
		</Field>

		<Field label={t('alerts.target')}>
			<Select bind:value={provider}>
				<option value="webhook">{t('alerts.webhook')}</option>
				{#if antenneAllowed}<option value="antenne">{t('alerts.antenneGlobal')}</option>{/if}
			</Select>
		</Field>

		{#if provider === 'webhook'}
			<div class="grid gap-4 sm:grid-cols-2">
				<Field
					label={t('alerts.threshold')}
					error={thresholdValid ? undefined : t('alerts.thresholdError')}
				>
					<Input type="number" min="1" bind:value={threshold} />
				</Field>
				<Field
					label={t('alerts.window')}
					error={windowValid ? undefined : t('alerts.windowError')}
				>
					<Input type="number" min="1" max="1440" bind:value={windowMinutes} />
				</Field>
			</div>

			<Field
				label={t('alerts.webhookUrl')}
				error={webhookUrl && !webhookValid ? t('alerts.webhookUrlError') : undefined}
			>
				<Input
					bind:value={webhookUrl}
					class="font-fc-mono"
					placeholder="https://antenne.facile.studio/hooks/…"
				/>
			</Field>

			<Field
				label={t('alerts.authHeader')}
				helper={t('alerts.authHeaderHint')}
				error={headerPairValid ? undefined : t('alerts.authHeaderError')}
			>
				<Input bind:value={webhookHeader} class="font-fc-mono" placeholder="X-Antenne-Token" />
			</Field>

			<SecretField bind:value={webhookSecret} label={t('alerts.authSecret')} editable mask="full" />
		{/if}
	</div>

	{#snippet footer()}
		<div class="flex justify-end gap-2">
			<Button size="lg" variant="ghost" onclick={() => (open = false)}>{t('common.cancel')}</Button>
			<Button size="lg" disabled={creating || !formValid} onclick={create}>
				{creating ? t('common.creating') : t('alerts.createRule')}
			</Button>
		</div>
	{/snippet}
</Drawer>

<ConfirmModal
	open={pending !== null}
	tone="danger"
	icon={icons.remove}
	title={t('alerts.deleteTitle')}
	description={t('alerts.deleteBody')}
	confirmLabel={t('common.delete')}
	onConfirm={remove}
	onCancel={() => (pending = null)}
/>