<script lang="ts">
	import { Badge } from '@facile/muse';
	import type { LogLevel } from '$lib/backend';
	import { levelTone } from '$lib/levels';

	let { level, class: className = '' }: { level: LogLevel; class?: string } = $props();
</script>

<Badge tone={levelTone(level)} class={`uppercase tabular-nums ${className}`}>{level}</Badge>
