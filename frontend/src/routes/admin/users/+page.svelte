<script lang="ts">
	import { auth } from '$lib/admin/auth.svelte';
	import PageHeader from '$lib/admin/components/PageHeader.svelte';
	import { i18n } from '$lib/i18n/state.svelte';
</script>

<section class="space-y-6">
	<PageHeader title={i18n.m.admin.users.heading} />
	<div class="card preset-filled-surface-50-950 border border-surface-200-800 p-6 space-y-3">
		<p class="text-sm opacity-80">{i18n.m.admin.users.iterationNotice}</p>
		{#if auth.user}
			<p class="text-sm opacity-70">{i18n.m.admin.users.currentSession(auth.user.email)}</p>
		{/if}
	</div>
</section>
