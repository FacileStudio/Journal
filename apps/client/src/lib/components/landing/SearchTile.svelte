<script lang="ts">
	import { Icon, icons } from '@facile/muse';
	import { Bar, Caret, Stage, Tile, grow, type Act } from '@facile/mise-en-scene';
	import LevelBadge from '$lib/components/LevelBadge.svelte';
	import type { LogLevel } from '$lib/backend';
	import { useT } from '$lib/i18n';

	const t = useT();

	const QUERY = 'timeout';

	const HITS: { app: string; level: LogLevel; before: string }[] = [
		{ app: 'nuage', level: 'error', before: 'upload ' },
		{ app: 'courrier', level: 'warn', before: 'smtp ' },
		{ app: 'agenda', level: 'error', before: 'caldav ' },
		{ app: 'plume', level: 'warn', before: 'export ' }
	];

	let typed = $state('');
	let found = $state(false);

	function reset() {
		typed = '';
		found = false;
	}

	function poster() {
		typed = QUERY;
		found = true;
	}

	async function script(act: Act) {
		await act.wait(700);
		await act.type(QUERY, (v) => (typed = v), 90);
		await act.wait(350);
		found = true;
		await act.wait(3200);
		reset();
		await act.wait(500);
	}
</script>

<Tile title={t('landing.cards.search.title')} body={t('landing.cards.search.body')}>
	<Stage width={320} height={240} {script} {reset} {poster}>
		{#snippet children()}
			<div class="panel absolute top-5 right-5 bottom-7 left-5 flex flex-col gap-3 rounded-fc-lg bg-fc-page p-3.5">
				<div class="flex items-center gap-2 rounded-fc-pill bg-fc-field px-3 py-1.5 text-fc-xs">
					<Icon icon={icons.search} size={14} class="shrink-0 text-fc-fg-muted" />
					{#if typed}
						<span class="font-fc-mono text-fc-fg">{typed}{#if !found}<Caret />{/if}</span>
					{:else}
						<span class="text-fc-fg-muted">{t('landing.demo.search')}</span>
					{/if}
				</div>
				<ul class="flex flex-col">
					{#if found}
						{#each HITS as hit (hit.app)}
							<li class="flex items-center gap-2 py-1.5 whitespace-nowrap" in:grow={{ duration: 320 }} out:grow={{ duration: 220 }}>
								<span class="shrink-0 rounded-fc-pill bg-fc-surface px-2 py-0.5 font-fc-mono text-[10px] text-fc-fg">{hit.app}</span>
								<LevelBadge level={hit.level} class="shrink-0" />
								<span class="min-w-0 truncate font-fc-mono text-[11px] text-fc-fg">
									{hit.before}<mark class="rounded-fc-xs bg-fc-warning/20 px-0.5 text-fc-fg">{QUERY}</mark>
								</span>
							</li>
						{/each}
					{:else}
						{#each [64, 48, 56, 44] as w, i (i)}
							<li class="flex items-center gap-2 py-2.5" out:grow={{ duration: 220 }}><Bar w="{w}%" /></li>
						{/each}
					{/if}
				</ul>
			</div>
		{/snippet}
	</Stage>
</Tile>

<style>
	.panel {
		box-shadow: 0 12px 24px -14px rgb(0 0 0 / 0.3);
	}
</style>
