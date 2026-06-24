<script lang="ts">
	import { onMount } from 'svelte';
	import { listRunners } from '$lib/api';
	import RunnerCard from '$lib/components/RunnerCard.svelte';
	import { i18n } from '$lib/i18n/state.svelte';
	import type { Runner } from '$lib/types';

	let runners: Runner[] = $state([]);
	let query = $state('');

	onMount(async () => {
		runners = await listRunners();
	});

	const filtered = $derived(
		query.trim().length === 0
			? runners
			: runners.filter((r) => r.name.toLowerCase().includes(query.toLowerCase()))
	);
</script>

<section class="space-y-6">
	<header>
		<h1 class="h2">{i18n.m.runnersList.heading}</h1>
		<p class="opacity-80 text-sm mt-1">{i18n.m.runnersList.description}</p>
	</header>

	<input
		type="search"
		placeholder={i18n.m.runnersList.searchPlaceholder}
		bind:value={query}
		class="input"
		aria-label={i18n.m.runnersList.searchLabel}
	/>

	{#if filtered.length === 0}
		<p class="opacity-70">{i18n.m.runnersList.noMatch(query)}</p>
	{:else}
		<ul class="grid gap-3 sm:grid-cols-2">
			{#each filtered as runner (runner.id)}
				<li><RunnerCard {runner} /></li>
			{/each}
		</ul>
	{/if}
</section>
