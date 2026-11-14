<script lang="ts">
	import { Badge } from '@facile/muse';
	import { spaces } from '$lib/workspaces.svelte';

	let { id }: { id: number } = $props();
</script>

{#if spaces.multi && spaces.name(id)}
	<Badge class="shrink-0">{spaces.name(id)}</Badge>
{/if}
