<script lang="ts">
	import { Icon, Switch, icons } from '@facile/muse';
	import { Roll, Stage, Tile, grow, type Act } from '@facile/mise-en-scene';
	import { levelFill, levelWeight } from '$lib/levels';
	import { useT } from '$lib/i18n';

	const t = useT();

	const INFO = [38, 44, 40, 52, 46, 50, 42, 48, 44, 40];
	const ERRORS = [4, 6, 3, 5, 4, 6, 5, 22, 34, 46];
	const CALM = 7;

	let errors = $state(ERRORS.map((v, i) => (i < CALM ? v : 0)));
	let fired = $state(false);

	function reset() {
		errors = ERRORS.map((v, i) => (i < CALM ? v : 0));
		fired = false;
	}

	function poster() {
		errors = [...ERRORS];
		fired = true;
	}

	async function script(act: Act) {
		await act.wait(900);
		for (let i = CALM; i < ERRORS.length; i++) {
			errors[i] = ERRORS[i];
			await act.wait(650);
		}
		fired = true;
		await act.wait(3000);
		reset();
		await act.wait(700);
	}
</script>

<Tile title={t('landing.cards.alert.title')} body={t('landing.cards.alert.body')}>
	<Stage width={320} height={240} {script} {reset} {poster}>
		{#snippet children()}
			<div class="panel absolute top-5 right-5 bottom-7 left-5 flex flex-col gap-3 rounded-fc-lg bg-fc-page p-3.5">
				<div class="flex h-20 items-end gap-1.5">
					{#each INFO as info, i (i)}
						<div class="flex flex-1 flex-col justify-end gap-px">
							<div
								class="bar rounded-t-fc-xs"
								style:height="{errors[i] * 0.9}px"
								style:background={levelFill('error')}
							></div>
							<div
								class="rounded-b-fc-xs"
								class:rounded-t-fc-xs={errors[i] === 0}
								style:height="{info * 0.5}px"
								style:background={levelFill('info')}
								style:opacity={levelWeight('info')}
							></div>
						</div>
					{/each}
				</div>
				<div class="flex items-center gap-3 rounded-fc-md bg-fc-component px-3 py-2.5">
					<div class="flex min-w-0 flex-1 flex-col gap-0.5">
						<span class="font-fc-mono text-fc-xs font-bold text-fc-fg">nuage-upload</span>
						<span class="truncate text-[10px] text-fc-fg-muted">
							<span class="font-fc-mono">level:error</span>
							· {t('alerts.conditionValue', { threshold: 20, minutes: 5 })}
						</span>
					</div>
					<Switch checked aria-label="nuage-upload" />
				</div>
				<div class="flex items-center justify-between gap-2 px-1 text-[10px] text-fc-fg-muted">
					<span>{t('alerts.lastFired')}</span>
					<Roll value={fired ? '14:02:11' : t('common.never')} class="h-4 w-16 text-right font-fc-mono leading-4" />
				</div>
			</div>
			{#if fired}
				<div
					class="note absolute top-3 right-3 flex items-center gap-2 rounded-fc-pill bg-fc-inverse py-1.5 pr-3 pl-2 text-fc-xs text-fc-on-inverse"
					in:grow={{ duration: 300 }}
					out:grow={{ duration: 220 }}
				>
					<Icon icon={icons.notification} size={14} class="shrink-0" />
					{t('landing.cards.alert.note')}
				</div>
			{/if}
		{/snippet}
	</Stage>
</Tile>

<style>
	.panel {
		box-shadow: 0 12px 24px -14px rgb(0 0 0 / 0.3);
	}
	.note {
		box-shadow: 0 10px 22px -10px rgb(0 0 0 / 0.4);
	}
	.bar {
		transition: height 0.5s cubic-bezier(0.25, 0.8, 0.25, 1);
	}
</style>
