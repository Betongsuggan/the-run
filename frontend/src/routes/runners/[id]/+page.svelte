<script lang="ts">
	import { page } from '$app/stores';
	import { resolve } from '$app/paths';
	import { getRunner, listResultsForRunner } from '$lib/api';
	import { i18n } from '$lib/i18n/state.svelte';
	import type { ResultExpanded, Runner } from '$lib/types';
	import ResultsTable from '$lib/components/ResultsTable.svelte';
	import SummaryStats from '$lib/components/SummaryStats.svelte';
	import PaceOverTimeChart from '$lib/components/PaceOverTimeChart.svelte';
	import DistanceDistributionChart from '$lib/components/DistanceDistributionChart.svelte';

	let runner: Runner | undefined = $state(undefined);
	let results: ResultExpanded[] = $state([]);
	let loading = $state(true);

	$effect(() => {
		const id = $page.params.id;
		if (!id) return;
		loading = true;
		(async () => {
			runner = await getRunner(id);
			results = await listResultsForRunner(id);
			loading = false;
		})();
	});
</script>

{#if loading}
	<p class="opacity-70">{i18n.m.runner.loading}</p>
{:else if !runner}
	<p>
		{i18n.m.runner.notFound}
		<a href={resolve('/runners')} class="text-primary-500 hover:underline"
			>{i18n.m.runner.backToRunners}</a
		>
	</p>
{:else}
	<section class="space-y-6">
		<header class="flex items-center gap-4">
			<div
				class="size-14 rounded-full preset-filled-primary-500 flex items-center justify-center font-bold text-xl"
			>
				{runner.name
					.split(' ')
					.map((p) => p[0])
					.slice(0, 2)
					.join('')}
			</div>
			<div>
				<h1 class="h2 leading-none">{runner.name}</h1>
				<div class="text-sm opacity-70 mt-1">
					{runner.gender}{#if runner.birthYear}
						· {i18n.m.runner.bornIn(runner.birthYear)}
					{/if}
				</div>
			</div>
		</header>

		<SummaryStats {results} />

		<section class="grid gap-4 lg:grid-cols-2">
			<div class="card preset-filled-surface-100-900 border border-surface-200-800 p-4">
				<h3 class="h5 mb-3">{i18n.m.runner.paceOverTime}</h3>
				<PaceOverTimeChart {results} />
			</div>
			<div class="card preset-filled-surface-100-900 border border-surface-200-800 p-4">
				<h3 class="h5 mb-3">{i18n.m.runner.distanceDistribution}</h3>
				<DistanceDistributionChart {results} />
			</div>
		</section>

		<section>
			<h2 class="h3 mb-3">{i18n.m.runner.resultsHeading}</h2>
			<ResultsTable {results} />
		</section>
	</section>
{/if}
