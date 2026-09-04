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
			error = err instanceof Error ? err.message : 'Failed to load alert rules';
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
			toast.success('Alert rule created.');
		} catch (err) {
			createError = err instanceof Error ? err.message : 'Could not create the rule';
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
			toast.danger(err instanceof Error ? err.message : 'Could not update the rule');
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
			toast.success(`Deleted “${rule.name}”.`);
		} catch (err) {
			toast.danger(err instanceof Error ? err.message : 'Could not delete the rule');
		} finally {
			pending = null;
		}
	}

	onMount(() => {
		if (!auth?.user?.is_admin) {
			void goto('/');
			return;
		}
		void load();
	});
</script>

<svelte:head><title>Alerts — Journal</title></svelte:head>

<PageHeader
	title="Alerts"
	description="A rule counts matches of a saved query over a rolling window and POSTs to a webhook when the count reaches the threshold."
>
	{#snippet actions()}
		<Button
			size="sm"
			icon={icons.plus}
			disabled={queries.length === 0}
			onclick={() => {
				reset();
				open = true;
			}}
		>
			New rule
		</Button>
	{/snippet}
</PageHeader>

{#if error}
	<Alert tone="danger" title="Could not load alert rules">{error}</Alert>
{:else if loading}
	<div class="flex justify-center py-16"><Spinner /></div>
{:else if queries.length === 0}
	<EmptyState
		icon={icons.filter}
		title="Save a query first"
		description="An alert rule watches a named filter set, so there is nothing to watch until one exists."
	>
		<Button href="/logs" icon={icons.history}>Open the explorer</Button>
	</EmptyState>
{:else if alerts.length === 0}
	<EmptyState
		icon={icons.notification}
		title="No alert rules yet"
		description="Pick a saved query, a threshold and a window, and Journal will call your webhook when the count is reached."
	>
		<Button icon={icons.plus} onclick={() => (open = true)}>New rule</Button>
	</EmptyState>
{:else}
	<Table>
		<thead>
			<tr>
				<th scope="col">Rule</th>
				<th scope="col">Condition</th>
				<th scope="col">Target</th>
				<th scope="col">Last fired</th>
				<th scope="col">Enabled</th>
				<th scope="col" aria-label="Actions"></th>
			</tr>
		</thead>
		<tbody>
			{#each alerts as rule (rule.id)}
				<tr>
					<td class="font-medium">{rule.name}</td>
					<td class="text-fc-fg-muted">
						<span class="font-fc-mono text-fc-xs">{rule.query_name}</span>
						· ≥ {rule.threshold} in {rule.window_minutes}m
					</td>
					<td class="font-fc-mono text-fc-xs text-fc-fg-muted">
						{#if rule.provider === 'antenne'}
							Antenne (global)
						{:else}
							{host(rule.webhook_url)}
						{/if}
					</td>
					<td class="whitespace-nowrap text-fc-fg-muted">
						{rule.last_fired_at ? formatDate(rule.last_fired_at) : 'never'}
					</td>
					<td>
						<Switch
							checked={rule.enabled}
							disabled={togglingId === rule.id}
							aria-label="Enable {rule.name}"
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
							Delete
						</Button>
					</td>
				</tr>
			{/each}
		</tbody>
	</Table>
{/if}

<Drawer
	bind:open
	title="New alert rule"
	description="Journal evaluates every rule once a minute and re-arms only after a full window."
	onClose={reset}
>
	<div class="flex flex-col gap-4">
		{#if createError}
			<Alert tone="danger" title="Could not create the rule">{createError}</Alert>
		{/if}

		<Field label="Name">
			<Input bind:value={name} placeholder="Nuage error spike" />
		</Field>

		<Field label="Saved query" helper="The filter set whose matches are counted.">
			<Select bind:value={savedQueryId}>
				<option value="" disabled>Pick a saved query…</option>
				{#each queries as saved (saved.id)}
					<option value={String(saved.id)}>{saved.name}</option>
				{/each}
			</Select>
		</Field>

		<Field label="Target">
			<Select bind:value={provider}>
				<option value="webhook">Webhook</option>
				<option value="antenne">Antenne (global)</option>
			</Select>
		</Field>

		{#if provider === 'webhook'}
			<div class="grid gap-4 sm:grid-cols-2">
				<Field
					label="Threshold"
					error={thresholdValid ? undefined : 'A whole number, 1 or more.'}
				>
					<Input type="number" min="1" bind:value={threshold} />
				</Field>
				<Field
					label="Window (minutes)"
					error={windowValid ? undefined : 'Between 1 and 1440.'}
				>
					<Input type="number" min="1" max="1440" bind:value={windowMinutes} />
				</Field>
			</div>

			<Field
				label="Webhook URL"
				error={webhookUrl && !webhookValid ? 'Must be an http or https URL.' : undefined}
			>
				<Input
					bind:value={webhookUrl}
					class="font-fc-mono"
					placeholder="https://antenne.facile.studio/hooks/…"
				/>
			</Field>

			<Field
				label="Auth header"
				helper="Optional. Name and secret are set together or not at all."
				error={headerPairValid ? undefined : 'Set both the header name and its secret.'}
			>
				<Input bind:value={webhookHeader} class="font-fc-mono" placeholder="X-Antenne-Token" />
			</Field>

			<SecretField bind:value={webhookSecret} label="Auth secret" editable mask="full" />
		{/if}
	</div>

	{#snippet footer()}
		<div class="flex justify-end gap-2">
			<Button size="lg" variant="ghost" onclick={() => (open = false)}>Cancel</Button>
			<Button size="lg" disabled={creating || !formValid} onclick={create}>
				{creating ? 'Creating…' : 'Create rule'}
			</Button>
		</div>
	{/snippet}
</Drawer>

<ConfirmModal
	open={pending !== null}
	tone="danger"
	icon={icons.remove}
	title="Delete this alert rule?"
	description="It stops firing immediately and its webhook is forgotten — nothing will tell you the next time the query matches."
	confirmLabel="Delete"
	onConfirm={remove}
	onCancel={() => (pending = null)}
/>