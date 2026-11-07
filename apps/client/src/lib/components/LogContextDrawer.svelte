<script lang="ts">
	import { Alert, Button, Drawer, Icon, Spinner, icons, toast } from '@facile/muse';
	import { useT } from '$lib/i18n';
	import { formatDate } from '$lib/format';

	const t = useT();

	type LogEntry = {
		id: number;
		app: string;
		level: string;
		message: string;
		meta?: Record<string, unknown>;
		created_at: string;
		received_at: string;
	};

	let { anchorId = $bindable<number | null>(null) }: { anchorId?: number | null } = $props();

	const open = $derived(anchorId !== null);

	let entry = $state<LogEntry | null>(null);
	let loading = $state(true);
	let error = $state<string | null>(null);

	$effect(() => {
		if (open && anchorId !== null) {
			loading = true;
			error = null;
			fetch(`/api/logs/${anchorId}`)
				.then((r) => (r.ok ? r.json() : Promise.reject(new Error(r.statusText))))
				.then((data: LogEntry) => {
					entry = data;
				})
				.catch((e: unknown) => {
					error = e instanceof Error ? e.message : t('drawer.loadFailed');
				})
				.finally(() => {
					loading = false;
				});
		} else if (!open) {
			entry = null;
		}
	});

	function onClose() {
		anchorId = null;
	}

	/** Renders the whole entry as a plain-text block, ready to paste into another tool. */
	function contextText(e: LogEntry): string {
		const lines = [
			`ID:           ${e.id}`,
			`App:          ${e.app}`,
			`Level:        ${e.level}`,
			`Created:      ${e.created_at}`,
			`Received:     ${e.received_at}`,
			`Message:      ${e.message}`
		];
		if (e.meta && Object.keys(e.meta).length > 0) {
			lines.push('', 'Meta:');
			for (const [key, value] of Object.entries(e.meta)) {
				let rendered: string;
				try {
					rendered = JSON.stringify(value);
				} catch {
					rendered = String(value);
				}
				lines.push(`  ${key}: ${rendered}`);
			}
		}
		return lines.join('\n');
	}

	function copyEntry() {
		if (!entry) return;
		const text = contextText(entry);
		if (navigator.clipboard && navigator.clipboard.writeText) {
			navigator.clipboard
				.writeText(text)
				.then(() => toast.success(t('clipboard.copied')))
				.catch(() => toast.danger(t('clipboard.failed')));
		} else {
			toast.danger(t('clipboard.unavailable'));
		}
	}
</script>

<Drawer
	{open}
	title={t('drawer.title')}
	description={entry ? `${entry.app} · ${entry.level}` : undefined}
	onClose={onClose}
>
	{#snippet footer()}
		{#if entry}
			<Button class="w-full" variant="outline" onclick={copyEntry}>
				<Icon icon={icons.copy} size={16} />
				{t('drawer.copyContext')}
			</Button>
		{/if}
	{/snippet}

	<div class="space-y-4 p-4">
		{#if loading}
			<div class="flex justify-center py-8">
				<Spinner />
			</div>
		{:else if error}
			<Alert tone="danger" title={t('drawer.loadFailed')}>{error}</Alert>
		{:else if entry}
			<dl class="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-sm">
				<dt class="text-fc-muted-foreground font-medium">ID</dt>
				<dd class="font-mono">{entry.id}</dd>
				<dt class="text-fc-muted-foreground font-medium">{t('common.app')}</dt>
				<dd>{entry.app}</dd>
				<dt class="text-fc-muted-foreground font-medium">{t('common.level')}</dt>
				<dd>{entry.level}</dd>
				<dt class="text-fc-muted-foreground font-medium">{t('drawer.created')}</dt>
				<dd>{formatDate(entry.created_at)}</dd>
				<dt class="text-fc-muted-foreground font-medium">{t('drawer.receivedLabel')}</dt>
				<dd>{formatDate(entry.received_at)}</dd>
				<dt class="text-fc-muted-foreground font-medium pt-2">{t('drawer.message')}</dt>
				<dd class="pt-2">{entry.message}</dd>
			</dl>

			{#if entry.meta && Object.keys(entry.meta).length > 0}
				<div>
					<h3 class="text-fc-muted-foreground mb-2 text-xs font-semibold tracking-wide uppercase">
						{t('drawer.meta')}
					</h3>
					<pre class="bg-fc-muted/40 text-fc-foreground overflow-auto rounded-lg p-3 text-xs"><code
							>{JSON.stringify(entry.meta, null, 2)}</code
						></pre>
				</div>
			{/if}
		{/if}
	</div>
</Drawer>