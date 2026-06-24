<script lang="ts">
	import { page } from '$app/stores';
	import { resolve } from '$app/paths';
	import { getResult } from '$lib/api';
	import { i18n } from '$lib/i18n/state.svelte';
	import type { ResultExpanded } from '$lib/types';
	import { formatDate, formatDistance, formatPace, formatTime } from '$lib/format';
	import SplitsTable from '$lib/components/SplitsTable.svelte';

	let result: ResultExpanded | undefined = $state(undefined);
	let loading = $state(true);

	$effect(() => {
		const id = $page.params.id;
		if (!id) return;
		loading = true;
		(async () => {
			result = await getResult(id);
			loading = false;
		})();
	});
</script>

{#if loading}
	<p class="opacity-70">{i18n.m.result.loading}</p>
{:else if !result}
	<p>
		{i18n.m.result.notFound}
		<a href={resolve('/runners')} class="text-primary-500 hover:underline"
			>{i18n.m.result.backToRunners}</a
		>
	</p>
{:else}
	<section class="space-y-6">
		<div class="text-sm">
			<a
				href={resolve('/runners/[id]', { id: result.runner.id })}
				class="text-primary-500 hover:underline">← {result.runner.name}</a
			>
		</div>

		<header class="space-y-1">
			<h1 class="h2">{result.race.name}</h1>
			<div class="opacity-80">
				{result.event.name} · {formatDate(result.event.date)}{#if result.event.location}
					· {result.event.location}
				{/if}
			</div>
		</header>

		<section class="grid gap-3 sm:grid-cols-4">
			<div class="card preset-filled-surface-100-900 border border-surface-200-800 p-4">
				<div class="text-xs opacity-70 uppercase">{i18n.m.result.distance}</div>
				<div class="text-xl font-bold">{formatDistance(result.race.distanceMeters)}</div>
			</div>
			<div class="card preset-filled-surface-100-900 border border-surface-200-800 p-4">
				<div class="text-xs opacity-70 uppercase">{i18n.m.result.finish}</div>
				<div class="text-xl font-bold font-mono">{formatTime(result.finishSeconds)}</div>
			</div>
			<div class="card preset-filled-surface-100-900 border border-surface-200-800 p-4">
				<div class="text-xs opacity-70 uppercase">{i18n.m.result.pace}</div>
				<div class="text-xl font-bold font-mono">
					{formatPace(result.race.distanceMeters, result.finishSeconds)}
				</div>
			</div>
			<div class="card preset-filled-surface-100-900 border border-surface-200-800 p-4">
				<div class="text-xs opacity-70 uppercase">{i18n.m.result.placement}</div>
				{#if result.placementOverall}
					<div class="text-xl font-bold">#{result.placementOverall}</div>
					{#if result.placementCategory && result.category.ageGroup}
						<div class="text-xs opacity-70 mt-0.5">
							{i18n.m.result.placementCategory(result.placementCategory, result.category.ageGroup)}
						</div>
					{/if}
				{:else}
					<div class="text-xl font-bold">—</div>
				{/if}
			</div>
		</section>

		<div class="text-sm opacity-80">
			{i18n.m.result.bib}
			<span class="font-mono">{result.bib}</span> · {i18n.m.result.category}
			{result.category.gender ?? '—'}{#if result.category.ageGroup}
				· {result.category.ageGroup}
			{/if}
		</div>

		<section>
			<h2 class="h4 mb-3">{i18n.m.result.splitsHeading}</h2>
			<SplitsTable splits={result.splits ?? []} />
		</section>

		{#if result.conditions}
			<section>
				<h2 class="h4 mb-2">{i18n.m.result.conditionsHeading}</h2>
				<p class="opacity-90">{result.conditions}</p>
			</section>
		{/if}

		{#if result.notes}
			<section>
				<h2 class="h4 mb-2">{i18n.m.result.notesHeading}</h2>
				<p class="opacity-90 whitespace-pre-wrap">{result.notes}</p>
			</section>
		{/if}
	</section>
{/if}
