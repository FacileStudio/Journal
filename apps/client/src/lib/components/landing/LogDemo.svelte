<script lang="ts">
	import { tick } from 'svelte';
	import { fade } from 'svelte/transition';
	import { Icon } from '@facile/muse';
	import { Bar, Caret, Cursor, Stage, Terminal, Window, cursorAt, grow, type Act } from '@facile/mise-en-scene';
	import LevelBadge from '$lib/components/LevelBadge.svelte';
	import { JOURNAL_ICON } from '$lib/icons';
	import { useT } from '$lib/i18n';
	import { COMMAND, FILTERS, INCOMING, ROWS, TAIL, shown, type Filter, type TailLine } from './demo';

	const t = useT();

	let typed = $state('');
	let tail = $state<TailLine[]>([]);
	let filter = $state<Filter>('all');
	let arrived = $state(false);
	let pressed = $state(false);
	let cursor = $state(cursorAt());
	let chip = $state<HTMLElement>();

	const rows = $derived(shown(arrived ? [INCOMING, ...ROWS] : ROWS, filter));

	function reset() {
		typed = '';
		tail = [];
		filter = 'all';
		arrived = false;
		pressed = false;
		cursor.visible = false;
	}

	function poster() {
		typed = COMMAND;
		tail = [...TAIL];
		filter = 'error';
		arrived = true;
	}

	async function stream(act: Act, from: number, to: number) {
		for (const line of TAIL.slice(from, to)) {
			tail.push(line);
			await act.wait(650);
		}
	}

	async function script(act: Act) {
		await act.wait(600);
		await act.type(COMMAND, (v) => (typed = v), 70);
		await act.wait(400);
		await stream(act, 0, 2);
		await tick();
		const target = act.point(chip, 0.88, 0.82);
		await act.move(cursor, { x: target.x + 90, y: target.y + 120 }, 0);
		cursor.visible = true;
		await act.move(cursor, target);
		await act.click(cursor, () => (pressed = true));
		pressed = false;
		filter = 'error';
		await act.wait(700);
		await stream(act, 2, 4);
		tail.push(TAIL[4]);
		arrived = true;
		await act.wait(2400);
		reset();
		await act.wait(600);
	}
</script>

{#snippet side()}
	<div class="ld-brand"><Icon icon={JOURNAL_ICON} size={18} /><b>Journal</b></div>
	<div class="ld-nav"><Bar w="58%" /></div>
	<div class="ld-nav ld-active">{t('nav.logs')}</div>
	<div class="ld-nav"><Bar w="44%" /></div>
	<div class="ld-nav"><Bar w="52%" /></div>
{/snippet}

<Stage
	width={960}
	height={520}
	compact={{ below: 640, width: 358, height: 590 }}
	label={t('landing.demo.label')}
	{script}
	{reset}
	{poster}
>
	{#snippet children({ compact })}
		<Window
			sidebar={compact ? undefined : side}
			style={compact ? 'top:12px;left:12px;width:334px;height:360px' : 'top:24px;left:48px;width:680px;height:420px'}
		>
			<div class="ld-body">
				<div class="ld-search">{t('landing.demo.search')}</div>
				<div class="ld-chips">
					{#each FILTERS as name (name)}
						{#if name === 'error'}
							<span bind:this={chip} class="ld-chip" class:ld-on={filter === name} class:ld-press={pressed}>{name}</span>
						{:else}
							<span class="ld-chip" class:ld-on={filter === name}>{name === 'all' ? t('landing.demo.all') : name}</span>
						{/if}
					{/each}
				</div>
				<ul class="ld-rows">
					{#each rows as row (row.id)}
						<li class="ld-row" class:ld-new={row.id === INCOMING.id} transition:grow={{ duration: 380 }}>
							{#if !compact}<span class="ld-time">{row.time}</span>{/if}
							{#if !compact}<span class="ld-app">{row.app}</span>{/if}
							<LevelBadge level={row.level} />
							{#if row.message}
								<span class="ld-msg">{row.message}</span>
							{:else}
								<span class="ld-msg"><Bar w="{row.width}%" /></span>
							{/if}
						</li>
					{/each}
				</ul>
			</div>
		</Window>
		<Terminal style={compact ? 'top:388px;left:12px;width:334px;height:160px' : 'top:276px;left:520px;width:392px;height:180px'}>
			<div class="ld-line"><span class="ld-dim">~ $</span> {typed}{#if tail.length === 0}<Caret />{/if}</div>
			{#each tail as line, i (i)}
				<div class="ld-line" in:fade={{ duration: 200 }}>
					{#if !compact}<span class="ld-dim">{line.time}</span>{/if}
					<span class="ld-dim">{line.app}</span>
					<span class="ld-level" data-level={line.level}>{line.level.toUpperCase()}</span>
					{line.message}
				</div>
			{/each}
		</Terminal>
		<Cursor {cursor} />
	{/snippet}
</Stage>

<style>
	.ld-brand {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 14px 18px 18px;
		font-family: var(--font-fc-title);
		font-size: 14px;
	}
	.ld-nav {
		display: flex;
		align-items: center;
		height: 34px;
		margin: 0 10px;
		padding: 0 10px;
		border-radius: 10px;
		font-size: 13px;
	}
	.ld-active {
		background: var(--color-fc-surface);
		font-weight: 700;
	}
	.ld-body {
		display: flex;
		flex-direction: column;
		min-height: 0;
		padding: 8px 18px 0;
		font-family: var(--font-fc-body);
	}
	.ld-search {
		padding: 9px 14px;
		border-radius: 999px;
		background: var(--color-fc-field);
		color: var(--color-fc-fg-muted);
		font-size: 13px;
	}
	.ld-chips {
		display: flex;
		gap: 6px;
		padding: 12px 0;
	}
	.ld-chip {
		padding: 4px 12px;
		border-radius: 999px;
		background: var(--color-fc-surface);
		color: var(--color-fc-fg-muted);
		font-family: var(--font-fc-mono);
		font-size: 12px;
		transition: all 0.2s ease;
	}
	.ld-on {
		background: var(--color-fc-accent);
		color: var(--color-fc-accent-fg);
	}
	.ld-press {
		scale: 0.94;
	}
	.ld-rows {
		margin: 0;
		padding: 0;
		list-style: none;
		overflow: hidden;
	}
	.ld-row {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 9px 8px;
		border-radius: 10px;
		white-space: nowrap;
		transition: background 0.4s ease;
	}
	.ld-new {
		background: color-mix(in oklab, var(--color-fc-danger) 10%, transparent);
	}
	.ld-time {
		color: var(--color-fc-fg-muted);
		font-family: var(--font-fc-mono);
		font-size: 12px;
	}
	.ld-app {
		padding: 2px 8px;
		border-radius: 999px;
		background: var(--color-fc-surface);
		font-family: var(--font-fc-mono);
		font-size: 12px;
	}
	.ld-row > :global(*) {
		flex: none;
	}
	.ld-row > .ld-msg {
		flex: 1;
		min-width: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		font-family: var(--font-fc-mono);
		font-size: 12px;
	}
	.ld-line {
		overflow: hidden;
		white-space: nowrap;
		text-overflow: ellipsis;
	}
	.ld-dim {
		color: var(--color-fc-on-inverse-muted);
	}
	.ld-level[data-level='warn'] {
		color: color-mix(in oklab, var(--color-fc-warning), var(--color-fc-on-inverse) 30%);
	}
	.ld-level[data-level='error'] {
		color: color-mix(in oklab, var(--color-fc-danger), var(--color-fc-on-inverse) 30%);
	}
	.ld-level[data-level='info'] {
		color: color-mix(in oklab, var(--color-fc-info), var(--color-fc-on-inverse) 30%);
	}
</style>
