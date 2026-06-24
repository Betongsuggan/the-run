<script lang="ts">
	import { onMount } from 'svelte';
	import Search from '@lucide/svelte/icons/search';
	import Users from '@lucide/svelte/icons/users';
	import { listRunners } from '$lib/api';
	import RunnerCard from '$lib/components/RunnerCard.svelte';
	import SectionHeader from '$lib/components/SectionHeader.svelte';
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
	<SectionHeader title={i18n.m.runnersList.heading} icon={Users} tone="secondary" />
	<p class="opacity-80 text-sm -mt-2">{i18n.m.runnersList.description}</p>

	<div class="space-y-2">
		<div class="relative">
			<Search
				class="absolute left-3 top-1/2 -translate-y-1/2 size-4 opacity-60 pointer-events-none"
				aria-hidden="true"
			/>
			<input
				type="search"
				placeholder={i18n.m.runnersList.searchPlaceholder}
				bind:value={query}
				class="input pl-10"
				aria-label={i18n.m.runnersList.searchLabel}
			/>
		</div>
		<div class="text-xs opacity-70">{i18n.m.runnersList.resultCount(filtered.length)}</div>
	</div>

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
