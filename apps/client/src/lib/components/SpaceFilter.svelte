<script lang="ts">
	import { Badge, Checkbox, Popover } from '@facile/muse';
	import { spaces } from '$lib/workspaces.svelte';
	import { useT } from '$lib/i18n';

	const t = useT();

	let open = $state(false);

	const label = $derived.by(() => {
		const chosen = spaces.selected;
		if (chosen.length === 0) return t('spaces.all');
		if (chosen.length === 1) return spaces.name(chosen[0]);
		return t('spaces.some', { count: chosen.length });
	});

	function shown(id: number) {
		return spaces.selected.length === 0 || spaces.selected.includes(id);
	}

	function toggle(id: number, on: boolean) {
		const current = spaces.selected.length ? spaces.selected : spaces.all.map((w) => w.id);
		const next = on ? [...current, id] : current.filter((other) => other !== id);
		if (next.length > 0) spaces.select(next);
	}
</script>

{#if spaces.all.length > 1}
	<Popover label={t('spaces.label')} bind:open align="auto" panelClass="w-72">
		{#snippet trigger()}
			<span class="max-w-48 truncate">{label}</span>
		{/snippet}
		<div class="flex flex-col gap-1 p-1">
			<p class="px-2 pb-1 text-fc-xs text-fc-fg-muted">{t('spaces.hint')}</p>
			<Checkbox
				class="min-h-11 rounded-fc-sm px-2 hover:bg-fc-surface"
				label={t('spaces.all')}
				checked={spaces.selected.length === 0}
				onchange={() => spaces.select([])}
			/>
			{#each spaces.all as workspace (workspace.id)}
				<div class="flex min-h-11 items-center gap-2 rounded-fc-sm px-2 hover:bg-fc-surface">
					<Checkbox
						class="min-w-0 flex-1"
						label={workspace.name}
						checked={shown(workspace.id)}
						onchange={(event) => toggle(workspace.id, event.currentTarget.checked)}
					/>
					{#if workspace.personal}<Badge>{t('spaces.personal')}</Badge>{/if}
				</div>
			{/each}
		</div>
	</Popover>
{/if}
