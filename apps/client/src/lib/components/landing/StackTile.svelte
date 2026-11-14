<script lang="ts">
	import { Roll, Stage, Tile, type Act } from '@facile/mise-en-scene';
	import LevelBadge from '$lib/components/LevelBadge.svelte';
	import { useT } from '$lib/i18n';

	const t = useT();

	const FRAMES = [
		{ raw: 'a — app.3f9c.js:1:48213', mapped: 'upload — src/upload.ts:42:7' },
		{ raw: 'r — app.3f9c.js:1:30977', mapped: 'send — src/queue.ts:18:3' },
		{ raw: 'o — app.3f9c.js:1:2291', mapped: 'drop — src/dropzone.ts:27:5' },
		{ raw: 'n — app.3f9c.js:1:1204', mapped: 'main — src/main.ts:9:1' }
	];

	let mapped = $state(0);

	function reset() {
		mapped = 0;
	}

	function poster() {
		mapped = FRAMES.length;
	}

	async function script(act: Act) {
		await act.wait(1400);
		for (let i = 0; i < FRAMES.length; i++) {
			mapped = i + 1;
			await act.wait(600);
		}
		await act.wait(3000);
		reset();
		await act.wait(700);
	}
</script>

<Tile title={t('landing.cards.stack.title')} body={t('landing.cards.stack.body')}>
	<Stage width={320} height={240} {script} {reset} {poster}>
		{#snippet children()}
			<div class="panel absolute top-5 right-5 bottom-7 left-5 flex flex-col gap-2.5 rounded-fc-lg bg-fc-surface p-3.5">
				<div class="flex items-center gap-2 whitespace-nowrap">
					<LevelBadge level="error" class="shrink-0" />
					<span class="min-w-0 truncate font-fc-mono text-[11px] text-fc-fg">TypeError: file is undefined</span>
				</div>
				<div class="flex flex-col gap-0.5 rounded-fc-sm bg-fc-bg p-3 font-fc-mono text-[10px] leading-4">
					{#each FRAMES as frame, i (frame.raw)}
						<Roll
							value={i < mapped ? frame.mapped : frame.raw}
							class="h-4 whitespace-nowrap {i < mapped ? 'text-fc-fg' : 'text-fc-fg-muted'}"
						/>
					{/each}
				</div>
				<p class="text-[10px] leading-4 text-fc-fg-muted">
					{t('table.frameMapped', { resolved: mapped, total: FRAMES.length })}
					<span class="font-fc-mono">v2.4.1</span>.
				</p>
			</div>
		{/snippet}
	</Stage>
</Tile>

<style>
	.panel {
		box-shadow: 0 12px 24px -14px rgb(0 0 0 / 0.3);
	}
</style>
