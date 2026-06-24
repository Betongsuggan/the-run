<script lang="ts">
	import { onMount } from 'svelte';
	import { getHello, type Hello } from '$lib/api';

	let state: { kind: 'loading' } | { kind: 'ok'; data: Hello } | { kind: 'err'; message: string } =
		$state({ kind: 'loading' });

	onMount(async () => {
		try {
			const data = await getHello();
			state = { kind: 'ok', data };
		} catch (e) {
			state = { kind: 'err', message: e instanceof Error ? e.message : String(e) };
		}
	});
</script>

<main>
	<h1>the-run</h1>

	{#if state.kind === 'loading'}
		<p>Loading…</p>
	{:else if state.kind === 'err'}
		<p style="color: crimson">Error: {state.message}</p>
	{:else}
		<p><strong>{state.data.message}</strong></p>
		<p><small>server time: {state.data.timestamp}</small></p>
	{/if}
</main>

<style>
	main {
		max-width: 40rem;
		margin: 4rem auto;
		padding: 0 1rem;
		font-family: system-ui, sans-serif;
	}
	h1 {
		margin-bottom: 1rem;
	}
</style>
