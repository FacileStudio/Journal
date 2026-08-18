<script lang="ts">
	import { onMount } from 'svelte';
	import { SecretField, SettingsRow, SettingsSection, StatusDot } from '@facile/muse';
	import { backend } from '$lib/backend';

	let registration = $state<boolean | null>(null);
	let reachable = $state<boolean | null>(null);

	const apiBase = $derived(
		typeof window === 'undefined' ? backend.baseUrl : `${window.location.origin}${backend.baseUrl}`
	);

	onMount(async () => {
		try {
			registration = (await backend.authConfig()).allow_registration;
			reachable = true;
		} catch {
			reachable = false;
		}
	});
</script>

<div class="flex flex-col gap-10">
	<SettingsSection title="Instance" description="Facts about the server this dashboard is talking to.">
		<SettingsRow label="API base" stacked>
			<SecretField value={apiBase} sensitive={false} />
		</SettingsRow>
		<SettingsRow label="Status" description="Whether the dashboard can reach the API right now.">
			{#if reachable === null}
				<StatusDot tone="neutral" label="Checking…" pulse />
			{:else if reachable}
				<StatusDot tone="success" label="Reachable" />
			{:else}
				<StatusDot tone="danger" label="Unreachable" />
			{/if}
		</SettingsRow>
		<SettingsRow
			label="Sign-ups"
			description="Set ALLOW_REGISTRATION=false on the server to lock this instance. The first account is always creatable."
		>
			{#if registration === null}
				<span class="text-fc-sm text-fc-fg-muted">—</span>
			{:else}
				<StatusDot
					tone={registration ? 'warning' : 'success'}
					label={registration ? 'Open' : 'Locked'}
				/>
			{/if}
		</SettingsRow>
	</SettingsSection>

	<SettingsSection
		title="Retention"
		description="Entries older than RETENTION_DAYS are deleted by an hourly job on the server; 0 keeps them forever. It is a deploy-time setting, not a dashboard one."
	>
		<SettingsRow
			label="Where to change it"
			description="Dokploy environment for this stack, alongside DATABASE_URL and CORS_ALLOWED_ORIGINS."
		>
			<span class="font-fc-mono text-fc-sm text-fc-fg-muted">RETENTION_DAYS</span>
		</SettingsRow>
	</SettingsSection>
</div>
