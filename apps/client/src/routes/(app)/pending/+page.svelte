<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { Alert, Button, EmptyState, Skeleton, icons, toast } from '@facile/muse';
	import { backend, ssoLoginUrl, type PendingAction } from '$lib/backend';
	import { rememberReturn } from '$lib/return-path';
	import { refusesOwnSession } from '$lib/session-refusal';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import PendingCard from '$lib/components/PendingCard.svelte';
	import { useT } from '$lib/i18n';

	const t = useT();

	let actions = $state<PendingAction[]>([]);
	let loading = $state(true);
	let busy = $state('');
	let notOwnSession = $state(false);

	const focused = $derived(page.url.searchParams.get('action') ?? '');
	const ordered = $derived([
		...actions.filter((action) => action.id === focused),
		...actions.filter((action) => action.id !== focused)
	]);

	onMount(load);

	/** Reports whether err is the refusal of a session no browser sign-in opened. */
	function refusedSession(err: unknown) {
		notOwnSession = refusesOwnSession(err);
		return notOwnSession;
	}

	/** Comes back to this page once the fresh browser sign-in returns. */
	function signInAgain() {
		rememberReturn(page.url.pathname + page.url.search);
	}

	/** The server's own words for any other 403, else the fallback text. */
	function reason(err: unknown, fallback: string): string {
		const refusal = err as { status?: unknown } | null;
		return refusal?.status === 403 && err instanceof Error ? err.message : fallback;
	}

	/** Loads the open actions. */
	async function load() {
		loading = true;
		try {
			actions = (await backend.listPending()).actions;
		} catch (err) {
			if (!refusedSession(err)) toast.danger(reason(err, t('pending.loadFailed')));
		}
		loading = false;
	}

	/** Runs one action once, then drops it from the list. */
	async function confirm(action: PendingAction) {
		busy = action.id;
		try {
			await backend.confirmPending(action.id);
			toast.success(t('pending.deleted'));
			actions = actions.filter((item) => item.id !== action.id);
		} catch (err) {
			if (!refusedSession(err)) {
				toast.danger(err instanceof Error ? err.message : t('pending.failed'));
				await load();
			}
		}
		busy = '';
	}

	/** Closes one action without running it. */
	async function dismiss(action: PendingAction) {
		busy = action.id;
		try {
			await backend.dismissPending(action.id);
			toast.success(t('pending.dismissed'));
			actions = actions.filter((item) => item.id !== action.id);
		} catch (err) {
			if (!refusedSession(err)) toast.danger(err instanceof Error ? err.message : t('pending.failed'));
		}
		busy = '';
	}
</script>

<svelte:head><title>{t('pending.title')} — Journal</title></svelte:head>

<PageHeader title={t('pending.title')} description={t('pending.intro')} />

<div class="mx-auto flex w-full max-w-2xl flex-col gap-4">
	{#if notOwnSession}
		<Alert tone="warning" title={t('pending.signInAgain')}>
			<p>{t('pending.signInAgainBody')}</p>
			<Button href={ssoLoginUrl} size="sm" class="mt-3" onclick={signInAgain}>{t('pending.signIn')}</Button>
		</Alert>
	{:else if loading}
		<Skeleton class="h-48 w-full" />
	{:else if actions.length === 0}
		<EmptyState icon={icons.check} title={t('pending.none')} description={t('pending.noneBody')} />
	{:else}
		{#each ordered as action (action.id)}
			<PendingCard {action} busy={busy === action.id} onconfirm={confirm} ondismiss={dismiss} />
		{/each}
	{/if}
</div>
