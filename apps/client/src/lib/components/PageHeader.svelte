<script lang="ts">
	import type { Snippet } from 'svelte';

	/*
	 * The charte's page-heading block (§4): title and description are one `gap-1` unit, so the
	 * description belongs to the title rather than floating between it and the body.
	 */
	let {
		title,
		description,
		actions,
		children
	}: {
		title: string;
		description?: string;
		actions?: Snippet;
		children?: Snippet;
	} = $props();
</script>

<div class="flex flex-wrap items-end justify-between gap-4">
	<div class="flex min-w-0 flex-col gap-1">
		<h1 class="text-fc-2xl font-semibold text-fc-fg">{title}</h1>
		{#if description}
			<p class="text-fc-sm text-fc-fg-muted">{description}</p>
		{/if}
		{#if children}{@render children()}{/if}
	</div>
	{#if actions}
		<div class="flex shrink-0 items-center gap-2">{@render actions()}</div>
	{/if}
</div>
