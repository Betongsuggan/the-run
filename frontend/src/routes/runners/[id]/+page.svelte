<script lang="ts">
	import { page } from '$app/stores';
	import { resolve } from '$app/paths';
	import Activity from '@lucide/svelte/icons/activity';
	import Route from '@lucide/svelte/icons/route';
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import Trophy from '@lucide/svelte/icons/trophy';
	import { getRunner, listResultsForRunner } from '$lib/api';
	import { i18n } from '$lib/i18n/state.svelte';
	import type { ResultExpanded, Runner } from '$lib/types';
	import ResultsTable from '$lib/components/ResultsTable.svelte';
	import SummaryStats from '$lib/components/SummaryStats.svelte';
	import PaceOverTimeChart from '$lib/components/PaceOverTimeChart.svelte';
	import DistanceDistributionChart from '$lib/components/DistanceDistributionChart.svelte';
	import SectionHeader from '$lib/components/SectionHeader.svelte';
	import Hero from '$lib/components/Hero.svelte';
	import ArchipelagoSilhouette from '$lib/illustrations/ArchipelagoSilhouette.svelte';

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

	function hashHue(s: string): number {
		let h = 0;
		for (let i = 0; i < s.length; i++) h = (h * 31 + s.charCodeAt(i)) | 0;
		return Math.abs(h) % 360;
	}

	const hue = $derived.by(() => {
		const r = runner;
		return r ? hashHue(r.id || r.name) : 0;
	});
	const initials = $derived.by(() => {
		const r = runner;
		if (!r) return '';
		return r.name
			.split(' ')
			.map((p: string) => p[0])
			.slice(0, 2)
			.join('');
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
		<a
			href={resolve('/runners')}
			class="inline-flex items-center gap-1 text-sm opacity-70 hover:opacity-100 transition-opacity"
		>
			<ArrowLeft class="size-4" />
			{i18n.m.runner.backToRunners}
		</a>

		<Hero class="!py-0">
			{#snippet background()}
				<ArchipelagoSilhouette
					class="absolute inset-x-0 bottom-0 h-20 w-full text-success-700 dark:text-success-300 opacity-70"
				/>
			{/snippet}
			<div class="flex items-center gap-5">
				<div
					class="size-16 sm:size-20 rounded-full flex items-center justify-center font-bold text-2xl sm:text-3xl text-white shadow-md shrink-0"
					style="background: oklch(0.6 0.15 {hue});"
					aria-hidden="true"
				>
					{initials}
				</div>
				<div class="min-w-0">
					<h1
						class="text-3xl sm:text-4xl font-semibold leading-tight tracking-tight truncate"
						style="font-family: var(--heading-font-family);"
					>
						{runner.name}
					</h1>
					<div class="text-sm opacity-75 mt-1">
						{runner.gender}{#if runner.birthYear} · {i18n.m.runner.bornIn(runner.birthYear)}{/if}
					</div>
				</div>
			</div>
		</Hero>

		<SummaryStats {results} />

		<section class="grid gap-4 lg:grid-cols-2">
			<div class="card preset-filled-surface-50-950 border border-surface-200-800 p-5">
				<div class="flex items-center gap-2 mb-3">
					<Activity class="size-4 text-primary-600 dark:text-primary-300" />
					<h3
						class="text-base font-semibold"
						style="font-family: var(--heading-font-family);"
					>
						{i18n.m.runner.paceOverTime}
					</h3>
				</div>
				<PaceOverTimeChart {results} />
			</div>
			<div class="card preset-filled-surface-50-950 border border-surface-200-800 p-5">
				<div class="flex items-center gap-2 mb-3">
					<Route class="size-4 text-success-600 dark:text-success-300" />
					<h3
						class="text-base font-semibold"
						style="font-family: var(--heading-font-family);"
					>
						{i18n.m.runner.distanceDistribution}
					</h3>
				</div>
				<DistanceDistributionChart {results} />
			</div>
		</section>

		<section>
			<SectionHeader title={i18n.m.runner.resultsHeading} icon={Trophy} tone="primary" />
			<ResultsTable {results} />
		</section>
	</section>
{/if}
