<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { auth } from '$lib/admin/auth.svelte';
	import LoginForm from '$lib/admin/components/LoginForm.svelte';
	import AdminShell from '$lib/admin/components/AdminShell.svelte';
	import { dataStore } from '$lib/store/data.svelte';

	let { children } = $props();

	const isAcceptRoute = $derived($page.url.pathname.startsWith('/admin/accept'));

	let hydrateError = $state<string | null>(null);

	onMount(async () => {
		// Probe /auth/me first so the LoginForm doesn't flash before we know
		// whether the cookie is still good.
		await auth.hydrate();
		if (auth.isAuthed) {
			dataStore.hydrate().catch((err) => {
				hydrateError = err instanceof Error ? err.message : String(err);
			});
		}
	});
</script>

{#if isAcceptRoute}
	{@render children()}
{:else if !auth.hydrated}
	<div class="mx-auto max-w-md py-12 text-center opacity-70">…</div>
{:else if !auth.isAuthed}
	<LoginForm />
{:else if hydrateError}
	<div class="card preset-tonal-error p-4 m-4">
		<p class="font-semibold">Failed to load data from the API.</p>
		<p class="text-sm opacity-80">{hydrateError}</p>
	</div>
{:else}
	<AdminShell>
		{@render children()}
	</AdminShell>
{/if}
