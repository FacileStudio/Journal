<script lang="ts">
	import { fade } from 'svelte/transition';
	import { Caret, Terminal } from '@facile/mise-en-scene';
	import type { TailLine } from './demo';

	interface Props {
		typed: string;
		tail: TailLine[];
		compact: boolean;
	}

	let { typed, tail, compact }: Props = $props();
</script>

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

<style>
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
